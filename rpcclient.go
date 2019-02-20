package net

import (
	"bytes"
	"encoding/gob"
	"errors"
	"github.com/golang/protobuf/proto"
	"github.com/vmihailenco/msgpack"
	"sync/atomic"
	"time"
)

type IRawRpcClient interface {
	ITcpClient
	CallCmd(cmd uint32, data []byte) ([]byte, error)
	CallCmdWithTimeout(cmd uint32, data []byte, timeout time.Duration) ([]byte, error)
}

type IRpcClient interface {
	CallCmd(cmd uint32, req interface{}, rsp interface{}) error
	CallCmdWithTimeout(cmd uint32, req interface{}, rsp interface{}, timeout time.Duration) error
	CallMethod(method string, req interface{}, rsp interface{}) error
	CallMethodWithTimeout(method string, req interface{}, rsp interface{}, timeout time.Duration) error
}

type RpcMessage struct {
	msg IMessage
	err error
}

type rpcsession struct {
	seq  int64
	done chan *RpcMessage
}

type RpcClient struct {
	*TcpClient
	sessionMap map[int64]*rpcsession
}

func (client *RpcClient) removeSession(seq int64) {
	client.Lock()
	delete(client.sessionMap, seq)
	client.Unlock()
}

func (client *RpcClient) CallCmd(cmd uint32, data []byte) ([]byte, error) {
	var session *rpcsession
	client.Lock()
	if client.running {
		session = &rpcsession{
			seq:  atomic.AddInt64(&client.sendSeq, 1),
			done: make(chan *RpcMessage, 1),
		}
		msg := NewRpcMessage(cmd, session.seq, data)
		select {
		case client.chSend <- asyncMessage{msg.Data(), nil}:
			client.sessionMap[session.seq] = session
		default:
			client.parent.OnSendQueueFull(client, msg)
			return nil, ErrRpcClientSendQueueIsFull
		}
	} else {
		client.Unlock()
		return nil, ErrRpcClientIsDisconnected
	}
	client.Unlock()
	defer client.removeSession(session.seq)
	msg, ok := <-session.done
	if !ok {
		return nil, ErrRpcCallClientError
	}
	return msg.msg.Body(), msg.err
}

func (client *RpcClient) CallCmdWithTimeout(cmd uint32, data []byte, timeout time.Duration) ([]byte, error) {
	var session *rpcsession
	client.Lock()
	if client.running {
		session = &rpcsession{
			seq:  atomic.AddInt64(&client.sendSeq, 1),
			done: make(chan *RpcMessage, 1),
		}
		msg := NewRpcMessage(cmd, session.seq, data)
		select {
		case client.chSend <- asyncMessage{msg.Data(), nil}:
			client.sessionMap[session.seq] = session
		default:
			client.parent.OnSendQueueFull(client, msg)
			return nil, ErrRpcClientSendQueueIsFull
		}
	} else {
		client.Unlock()
		return nil, ErrRpcClientIsDisconnected
	}
	client.Unlock()
	select {
	case msg, ok := <-session.done:
		if !ok {
			return nil, ErrRpcCallClientError
		}
		return msg.msg.Body(), msg.err
	case <-time.After(timeout):
		return nil, ErrRpcCallTimeout
	}
	return nil, ErrRpcCallClientError
}

func NewRpcClient(addr string, onConnected func(ITcpClient)) (IRawRpcClient, error) {
	engine := NewTcpEngine().(*TcpEngin)
	engine.running = true
	engine.SetSockRecvBlockTime(_conf_sock_rpc_recv_block_time)
	client, err := NewTcpClient(addr, engine, nil, true, onConnected)
	if err != nil {
		return nil, err
	}
	safeGo(func() {
		client.Keepalive(_conf_sock_keepalive_time)
	})
	rpcclient := &RpcClient{client, map[int64]*rpcsession{}}
	rpcclient.OnClose("-", func(ITcpClient) {
		rpcclient.Lock()
		defer rpcclient.Unlock()
		for _, session := range rpcclient.sessionMap {
			close(session.done)
		}
		rpcclient.sessionMap = map[int64]*rpcsession{}
	})

	engine.HandleOnMessage(func(c ITcpClient, msg IMessage) {
		//if engine.running {
		rpcclient.Lock()
		session, ok := rpcclient.sessionMap[msg.RpcSeq()]
		rpcclient.Unlock()
		if ok {
			if msg.Cmd() == CmdRpcMethodError {
				session.done <- &RpcMessage{msg, errors.New(string(msg.Body()))}
			} else {
				session.done <- &RpcMessage{msg, nil}
			}
		} else {
			if msg.Cmd() != CmdPing {
				logDebug("no rpcsession waiting for rpc response, cmd %X, ip: %v", msg.Cmd(), client.Ip())
			}
		}
		// } else {
		// 	logDebug("engine is not running, ignore rpc cmd %X, ip: %v", msg.Cmd(), client.Ip())
		// }
	})

	return rpcclient, nil
}

type JsonRpcClient struct {
	Client IRawRpcClient
}

func (client *JsonRpcClient) CallCmd(cmd uint32, req interface{}, rsp interface{}) error {
	data, err := json.Marshal(req)
	if err != nil {
		return err
	}
	rspdata, err := client.Client.CallCmd(cmd, data)
	if err != nil {
		return err
	}
	if rsp != nil {
		err = json.Unmarshal(rspdata, rsp)
	}
	return err
}

func (client *JsonRpcClient) CallCmdWithTimeout(cmd uint32, req interface{}, rsp interface{}, timeout time.Duration) error {
	data, err := json.Marshal(req)
	if err != nil {
		return err
	}
	rspdata, err := client.Client.CallCmdWithTimeout(cmd, data, timeout)
	if err != nil {
		return err
	}
	if rsp != nil {
		err = json.Unmarshal(rspdata, rsp)
	}
	return err
}

func (client *JsonRpcClient) CallMethod(method string, req interface{}, rsp interface{}) error {
	data, err := json.Marshal(req)
	if err != nil {
		return err
	}
	data = append(data, make([]byte, len(method)+1)...)
	copy(data[len(data)-len(method)-1:], method)
	data[len(data)-1] = byte(len(method))
	rspdata, err := client.Client.CallCmd(CmdRpcMethod, data)
	if err != nil {
		return err
	}
	if rsp != nil {
		err = json.Unmarshal(rspdata, rsp)
	}
	return err
}

func (client *JsonRpcClient) CallMethodWithTimeout(method string, req interface{}, rsp interface{}, timeout time.Duration) error {
	data, err := json.Marshal(req)
	if err != nil {
		return err
	}
	data = append(data, make([]byte, len(method)+1)...)
	copy(data[len(data)-len(method)-1:], method)
	data[len(data)-1] = byte(len(method))
	rspdata, err := client.Client.CallCmdWithTimeout(CmdRpcMethod, data, timeout)
	if err != nil {
		return err
	}
	if rsp != nil {
		err = json.Unmarshal(rspdata, rsp)
	}
	return err
}

func NewJsonRpcClient(addr string, onConnected func(ITcpClient)) (IRpcClient, error) {
	c, err := NewRpcClient(addr, nil)
	if err != nil {
		return nil, err
	}
	return &JsonRpcClient{c}, nil
}

type GobRpcClient struct {
	Client IRawRpcClient
}

func (client *GobRpcClient) CallCmd(cmd uint32, req interface{}, rsp interface{}) error {
	buffer := &bytes.Buffer{}
	err := gob.NewEncoder(buffer).Encode(req)
	if err != nil {
		return err
	}
	rspdata, err := client.Client.CallCmd(cmd, buffer.Bytes())
	if err != nil {
		return err
	}
	if rsp != nil {
		gob.NewDecoder(bytes.NewBuffer(rspdata)).Decode(rsp)
	}
	return err
}

func (client *GobRpcClient) CallCmdWithTimeout(cmd uint32, req interface{}, rsp interface{}, timeout time.Duration) error {
	buffer := &bytes.Buffer{}
	err := gob.NewEncoder(buffer).Encode(req)
	if err != nil {
		return err
	}
	rspdata, err := client.Client.CallCmdWithTimeout(cmd, buffer.Bytes(), timeout)
	if err != nil {
		return err
	}
	if rsp != nil {
		gob.NewDecoder(bytes.NewBuffer(rspdata)).Decode(rsp)
	}
	return err
}

func (client *GobRpcClient) CallMethod(method string, req interface{}, rsp interface{}) error {
	buffer := &bytes.Buffer{}
	err := gob.NewEncoder(buffer).Encode(req)
	if err != nil {
		return err
	}
	data := append(buffer.Bytes(), make([]byte, len(method)+1)...)
	copy(data[len(data)-len(method)-1:], method)
	data[len(data)-1] = byte(len(method))
	rspdata, err := client.Client.CallCmd(CmdRpcMethod, data)
	if err != nil {
		return err
	}
	if rsp != nil {
		gob.NewDecoder(bytes.NewBuffer(rspdata)).Decode(rsp)
	}
	return err
}

func (client *GobRpcClient) CallMethodWithTimeout(method string, req interface{}, rsp interface{}, timeout time.Duration) error {
	buffer := &bytes.Buffer{}
	err := gob.NewEncoder(buffer).Encode(req)
	if err != nil {
		return err
	}
	data := append(buffer.Bytes(), make([]byte, len(method)+1)...)
	copy(data[len(data)-len(method)-1:], method)
	data[len(data)-1] = byte(len(method))
	rspdata, err := client.Client.CallCmdWithTimeout(CmdRpcMethod, data, timeout)
	if err != nil {
		return err
	}
	if rsp != nil {
		gob.NewDecoder(bytes.NewBuffer(rspdata)).Decode(rsp)
	}
	return err
}

func NewGobRpcClient(addr string, onConnected func(ITcpClient)) (IRpcClient, error) {
	c, err := NewRpcClient(addr, nil)
	if err != nil {
		return nil, err
	}
	return &GobRpcClient{c}, nil
}

type MsgPackRpcClient struct {
	Client IRawRpcClient
}

func (client *MsgPackRpcClient) CallCmd(cmd uint32, req interface{}, rsp interface{}) error {
	data, err := msgpack.Marshal(req)
	if err != nil {
		return err
	}
	rspdata, err := client.Client.CallCmd(cmd, data)
	if err != nil {
		return err
	}
	if rsp != nil {
		err = msgpack.Unmarshal(rspdata, rsp)
	}
	return err
}

func (client *MsgPackRpcClient) CallCmdWithTimeout(cmd uint32, req interface{}, rsp interface{}, timeout time.Duration) error {
	data, err := msgpack.Marshal(req)
	if err != nil {
		return err
	}
	rspdata, err := client.Client.CallCmdWithTimeout(cmd, data, timeout)
	if err != nil {
		return err
	}
	if rsp != nil {
		err = msgpack.Unmarshal(rspdata, rsp)
	}
	return err
}

func (client *MsgPackRpcClient) CallMethod(method string, req interface{}, rsp interface{}) error {
	data, err := msgpack.Marshal(req)
	if err != nil {
		return err
	}
	data = append(data, make([]byte, len(method)+1)...)
	copy(data[len(data)-len(method)-1:], method)
	data[len(data)-1] = byte(len(method))
	rspdata, err := client.Client.CallCmd(CmdRpcMethod, data)
	if err != nil {
		return err
	}
	if rsp != nil {
		err = msgpack.Unmarshal(rspdata, rsp)
	}
	return err
}

func (client *MsgPackRpcClient) CallMethodWithTimeout(method string, req interface{}, rsp interface{}, timeout time.Duration) error {
	data, err := msgpack.Marshal(req)
	if err != nil {
		return err
	}
	data = append(data, make([]byte, len(method)+1)...)
	copy(data[len(data)-len(method)-1:], method)
	data[len(data)-1] = byte(len(method))
	rspdata, err := client.Client.CallCmdWithTimeout(CmdRpcMethod, data, timeout)
	if err != nil {
		return err
	}
	if rsp != nil {
		err = msgpack.Unmarshal(rspdata, rsp)
	}
	return err
}

func NewMsgPackRpcClient(addr string, onConnected func(ITcpClient)) (IRpcClient, error) {
	c, err := NewRpcClient(addr, nil)
	if err != nil {
		return nil, err
	}
	return &MsgPackRpcClient{c}, nil
}

type ProtobufRpcClient struct {
	Client IRawRpcClient
}

func (client *ProtobufRpcClient) CallCmd(cmd uint32, req interface{}, rsp interface{}) error {
	pbreq, ok := req.(proto.Message)
	if !ok {
		return errors.New("invalid req type: not pb.Message")
	}

	data, err := proto.Marshal(pbreq)
	if err != nil {
		return err
	}
	rspdata, err := client.Client.CallCmd(cmd, data)
	if err != nil {
		return err
	}
	if rsp != nil {
		pbrsp, ok := rsp.(proto.Message)
		if !ok {
			return errors.New("invalid rsp type: not pb.Message")
		}
		err = proto.Unmarshal(rspdata, pbrsp)
	}
	return err
}

func (client *ProtobufRpcClient) CallCmdWithTimeout(cmd uint32, req interface{}, rsp interface{}, timeout time.Duration) error {
	pbreq, ok := req.(proto.Message)
	if !ok {
		return errors.New("invalid req type: not pb.Message")
	}

	data, err := proto.Marshal(pbreq)
	if err != nil {
		return err
	}
	rspdata, err := client.Client.CallCmdWithTimeout(cmd, data, timeout)
	if err != nil {
		return err
	}
	if rsp != nil {
		pbrsp, ok := rsp.(proto.Message)
		if !ok {
			return errors.New("invalid rsp type: not pb.Message")
		}
		err = proto.Unmarshal(rspdata, pbrsp)
	}
	return err
}

func (client *ProtobufRpcClient) CallMethod(method string, req interface{}, rsp interface{}) error {
	pbreq, ok := req.(proto.Message)
	if !ok {
		return errors.New("invalid req type: not pb.Message")
	}

	data, err := proto.Marshal(pbreq)
	if err != nil {
		return err
	}
	data = append(data, make([]byte, len(method)+1)...)
	copy(data[len(data)-len(method)-1:], method)
	data[len(data)-1] = byte(len(method))
	rspdata, err := client.Client.CallCmd(CmdRpcMethod, data)
	if err != nil {
		return err
	}
	if rsp != nil {
		pbrsp, ok := rsp.(proto.Message)
		if !ok {
			return errors.New("invalid rsp type: not pb.Message")
		}
		err = proto.Unmarshal(rspdata, pbrsp)
	}
	return err
}

func (client *ProtobufRpcClient) CallMethodWithTimeout(method string, req interface{}, rsp interface{}, timeout time.Duration) error {
	pbreq, ok := req.(proto.Message)
	if !ok {
		return errors.New("invalid req type: not pb.Message")
	}

	data, err := proto.Marshal(pbreq)
	if err != nil {
		return err
	}
	data = append(data, make([]byte, len(method)+1)...)
	copy(data[len(data)-len(method)-1:], method)
	data[len(data)-1] = byte(len(method))
	rspdata, err := client.Client.CallCmdWithTimeout(CmdRpcMethod, data, timeout)
	if err != nil {
		return err
	}
	if rsp != nil {
		pbrsp, ok := rsp.(proto.Message)
		if !ok {
			return errors.New("invalid rsp type: not pb.Message")
		}
		err = proto.Unmarshal(rspdata, pbrsp)
	}
	return err
}

func NewProtobufRpcClient(addr string, onConnected func(ITcpClient)) (IRpcClient, error) {
	c, err := NewRpcClient(addr, nil)
	if err != nil {
		return nil, err
	}
	return &ProtobufRpcClient{c}, nil
}
