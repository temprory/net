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

type IRpcCodec interface {
	Marshal(v interface{}) ([]byte, error)
	Unmarshal(data []byte, v interface{}) error
}

// type IRawRpcClient interface {
// 	ITcpClient
// 	CallCmd(cmd uint32, req interface{}, rsp interface{}) error
// 	CallCmdWithTimeout(cmd uint32, req interface{}, rsp interface{}, timeout time.Duration) error
// 	CallMethod(method string, req interface{}, rsp interface{}) error
// 	CallMethodWithTimeout(method string, req interface{}, rsp interface{}, timeout time.Duration) error
// }

type IRpcClient interface {
	//ITcpClient
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
	codec      IRpcCodec
}

func (client *RpcClient) removeSession(seq int64) {
	client.Lock()
	delete(client.sessionMap, seq)
	if len(client.sessionMap) == 0 {
		client.sessionMap = map[int64]*rpcsession{}
	}
	client.Unlock()
}

func (client *RpcClient) callCmd(cmd uint32, data []byte) ([]byte, error) {
	var session *rpcsession
	client.Lock()
	if client.running {
		session = &rpcsession{
			seq:  atomic.AddInt64(&client.sendSeq, 1),
			done: make(chan *RpcMessage, 1),
		}
		msg := NewRpcMessage(cmd, session.seq, data)
		client.chSend <- asyncMessage{msg.data, nil}
		client.sessionMap[session.seq] = session
	} else {
		client.Unlock()
		return nil, ErrRpcClientIsDisconnected
	}
	client.Unlock()
	defer client.removeSession(session.seq)
	msg, ok := <-session.done
	if !ok {
		return nil, ErrRpcClientIsDisconnected
	}
	return msg.msg.Body(), msg.err
}

func (client *RpcClient) callCmdWithTimeout(cmd uint32, data []byte, timeout time.Duration) ([]byte, error) {
	var session *rpcsession
	client.Lock()
	if client.running {
		session = &rpcsession{
			seq:  atomic.AddInt64(&client.sendSeq, 1),
			done: make(chan *RpcMessage, 1),
		}
		msg := NewRpcMessage(cmd, session.seq, data)
		select {
		case client.chSend <- asyncMessage{msg.data, nil}:
			client.sessionMap[session.seq] = session
		case <-time.After(timeout):
			client.Unlock()
			return nil, ErrRpcCallTimeout
		}
	} else {
		client.Unlock()
		return nil, ErrRpcClientIsDisconnected
	}
	client.Unlock()
	defer client.removeSession(session.seq)
	select {
	case msg, ok := <-session.done:
		if !ok {
			return nil, ErrRpcClientIsDisconnected
		}
		return msg.msg.Body(), msg.err
	case <-time.After(timeout):
		return nil, ErrRpcCallTimeout
	}
	return nil, ErrRpcCallClientError
}

func (client *RpcClient) CallCmd(cmd uint32, req interface{}, rsp interface{}) error {
	data, err := client.codec.Marshal(req)
	if err != nil {
		return err
	}
	rspdata, err := client.callCmd(cmd, data)
	if err != nil {
		return err
	}
	if rsp != nil {
		err = client.codec.Unmarshal(rspdata, rsp)
	}
	return err
}

func (client *RpcClient) CallCmdWithTimeout(cmd uint32, req interface{}, rsp interface{}, timeout time.Duration) error {
	data, err := client.codec.Marshal(req)
	if err != nil {
		return err
	}
	rspdata, err := client.callCmdWithTimeout(cmd, data, timeout)
	if err != nil {
		return err
	}
	if rsp != nil {
		err = client.codec.Unmarshal(rspdata, rsp)
	}
	return err
}

func (client *RpcClient) CallMethod(method string, req interface{}, rsp interface{}) error {
	data, err := client.codec.Marshal(req)
	if err != nil {
		return err
	}
	data = append(data, make([]byte, len(method)+1)...)
	copy(data[len(data)-len(method)-1:], method)
	data[len(data)-1] = byte(len(method))
	rspdata, err := client.callCmd(CmdRpcMethod, data)
	if err != nil {
		return err
	}
	if rsp != nil {
		err = client.codec.Unmarshal(rspdata, rsp)
	}
	return err
}

func (client *RpcClient) CallMethodWithTimeout(method string, req interface{}, rsp interface{}, timeout time.Duration) error {
	data, err := client.codec.Marshal(req)
	if err != nil {
		return err
	}
	data = append(data, make([]byte, len(method)+1)...)
	copy(data[len(data)-len(method)-1:], method)
	data[len(data)-1] = byte(len(method))
	rspdata, err := client.callCmdWithTimeout(CmdRpcMethod, data, timeout)
	if err != nil {
		return err
	}
	if rsp != nil {
		err = client.codec.Unmarshal(rspdata, rsp)
	}
	return err
}

func NewRpcClient(addr string, engine ITcpEngin, codec IRpcCodec) (IRpcClient, error) {
	if engine == nil {
		engine = NewTcpEngine()
	}
	engine.SetSendQueueSize(_conf_sock_rpc_send_q_size)
	engine.SetSockRecvBlockTime(_conf_sock_rpc_recv_block_time)
	client, err := NewTcpClient(addr, engine, nil, true, nil)
	if err != nil {
		return nil, err
	}
	safeGo(func() {
		client.Keepalive(_conf_sock_keepalive_time)
	})
	rpcclient := &RpcClient{client, map[int64]*rpcsession{}, codec}
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

// func NewRpcClientWithCodec(addr string, codec IRpcCodec, engine ITcpEngin) (IRpcClient, error) {
// 	if codec == nil {
// 		logPanic("NewRpcClientWithCodec failed, invalid codec(nil)")
// 	}
// 	c, err := NewRpcClient(addr, engine, codec)
// 	return c, err
// }

type JsonRpcClient struct {
	*RpcClient
}

func (client *JsonRpcClient) CallCmd(cmd uint32, req interface{}, rsp interface{}) error {
	data, err := json.Marshal(req)
	if err != nil {
		return err
	}
	rspdata, err := client.callCmd(cmd, data)
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
	rspdata, err := client.callCmdWithTimeout(cmd, data, timeout)
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
	rspdata, err := client.callCmd(CmdRpcMethod, data)
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
	rspdata, err := client.callCmdWithTimeout(CmdRpcMethod, data, timeout)
	if err != nil {
		return err
	}
	if rsp != nil {
		err = json.Unmarshal(rspdata, rsp)
	}
	return err
}

func NewJsonRpcClient(addr string, engine ITcpEngin) (IRpcClient, error) {
	c, err := NewRpcClient(addr, engine, nil)
	if err != nil {
		return nil, err
	}
	return &JsonRpcClient{c.(*RpcClient)}, nil
}

type GobRpcClient struct {
	*RpcClient
}

func (client *GobRpcClient) CallCmd(cmd uint32, req interface{}, rsp interface{}) error {
	buffer := &bytes.Buffer{}
	err := gob.NewEncoder(buffer).Encode(req)
	if err != nil {
		return err
	}
	rspdata, err := client.callCmd(cmd, buffer.Bytes())
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
	rspdata, err := client.callCmdWithTimeout(cmd, buffer.Bytes(), timeout)
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
	rspdata, err := client.callCmd(CmdRpcMethod, data)
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
	rspdata, err := client.callCmdWithTimeout(CmdRpcMethod, data, timeout)
	if err != nil {
		return err
	}
	if rsp != nil {
		gob.NewDecoder(bytes.NewBuffer(rspdata)).Decode(rsp)
	}
	return err
}

func NewGobRpcClient(addr string, engine ITcpEngin) (IRpcClient, error) {
	c, err := NewRpcClient(addr, engine, nil)
	if err != nil {
		return nil, err
	}
	return &GobRpcClient{c.(*RpcClient)}, nil
}

type MsgpackRpcClient struct {
	*RpcClient
}

func (client *MsgpackRpcClient) CallCmd(cmd uint32, req interface{}, rsp interface{}) error {
	data, err := msgpack.Marshal(req)
	if err != nil {
		return err
	}
	rspdata, err := client.callCmd(cmd, data)
	if err != nil {
		return err
	}
	if rsp != nil {
		err = msgpack.Unmarshal(rspdata, rsp)
	}
	return err
}

func (client *MsgpackRpcClient) CallCmdWithTimeout(cmd uint32, req interface{}, rsp interface{}, timeout time.Duration) error {
	data, err := msgpack.Marshal(req)
	if err != nil {
		return err
	}
	rspdata, err := client.callCmdWithTimeout(cmd, data, timeout)
	if err != nil {
		return err
	}
	if rsp != nil {
		err = msgpack.Unmarshal(rspdata, rsp)
	}
	return err
}

func (client *MsgpackRpcClient) CallMethod(method string, req interface{}, rsp interface{}) error {
	data, err := msgpack.Marshal(req)
	if err != nil {
		return err
	}
	data = append(data, make([]byte, len(method)+1)...)
	copy(data[len(data)-len(method)-1:], method)
	data[len(data)-1] = byte(len(method))
	rspdata, err := client.callCmd(CmdRpcMethod, data)
	if err != nil {
		return err
	}
	if rsp != nil {
		err = msgpack.Unmarshal(rspdata, rsp)
	}
	return err
}

func (client *MsgpackRpcClient) CallMethodWithTimeout(method string, req interface{}, rsp interface{}, timeout time.Duration) error {
	data, err := msgpack.Marshal(req)
	if err != nil {
		return err
	}
	data = append(data, make([]byte, len(method)+1)...)
	copy(data[len(data)-len(method)-1:], method)
	data[len(data)-1] = byte(len(method))
	rspdata, err := client.callCmdWithTimeout(CmdRpcMethod, data, timeout)
	if err != nil {
		return err
	}
	if rsp != nil {
		err = msgpack.Unmarshal(rspdata, rsp)
	}
	return err
}

func NewMsgpackRpcClient(addr string, engine ITcpEngin) (IRpcClient, error) {
	c, err := NewRpcClient(addr, engine, nil)
	if err != nil {
		return nil, err
	}
	return &MsgpackRpcClient{c.(*RpcClient)}, nil
}

type ProtobufRpcClient struct {
	*RpcClient
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
	rspdata, err := client.callCmd(cmd, data)
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
	rspdata, err := client.callCmdWithTimeout(cmd, data, timeout)
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
	rspdata, err := client.callCmd(CmdRpcMethod, data)
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
	rspdata, err := client.callCmdWithTimeout(CmdRpcMethod, data, timeout)
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

func NewProtobufRpcClient(addr string, engine ITcpEngin) (IRpcClient, error) {
	c, err := NewRpcClient(addr, engine, nil)
	if err != nil {
		return nil, err
	}
	return &ProtobufRpcClient{c.(*RpcClient)}, nil
}
