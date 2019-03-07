package net

import (
	"errors"
	"sync/atomic"
	"time"
)

type IRpcClient interface {
	ITcpClient
	Codec() IRpcCodec
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

func (client *RpcClient) Codec() IRpcCodec {
	return client.codec
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

func NewRpcClient(addr string, engine ITcpEngin, codec IRpcCodec, onConnected func(ITcpClient)) (IRpcClient, error) {
	if engine == nil {
		engine = NewTcpEngine()
	}
	engine.SetSendQueueSize(_conf_sock_rpc_send_q_size)
	engine.SetSockRecvBlockTime(_conf_sock_rpc_recv_block_time)
	client, err := NewTcpClient(addr, engine, nil, true, onConnected)
	if err != nil {
		return nil, err
	}
	safeGo(func() {
		client.Keepalive(_conf_sock_keepalive_time)
	})

	if codec == nil {
		codec = DefaultRpcCodec
		logDebug("use default rpc codec: %v", defaultRpcCodecType)
	}
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
			if msg.Cmd() == CmdRpcError {
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

func NewGobRpcClient(addr string, engine ITcpEngin, onConnected func(ITcpClient)) (IRpcClient, error) {
	c, err := NewRpcClient(addr, engine, &RpcCodecGob{}, onConnected)
	return c, err
}

func NewJsonRpcClient(addr string, engine ITcpEngin, onConnected func(ITcpClient)) (IRpcClient, error) {
	c, err := NewRpcClient(addr, engine, &RpcCodecJson{}, onConnected)
	return c, err
}

func NewMsgpackRpcClient(addr string, engine ITcpEngin, onConnected func(ITcpClient)) (IRpcClient, error) {
	c, err := NewRpcClient(addr, engine, &RpcCodecMsgpack{}, onConnected)
	return c, err
}

func NewProtobufRpcClient(addr string, engine ITcpEngin, onConnected func(ITcpClient)) (IRpcClient, error) {
	return NewRpcClient(addr, engine, &RpcCodecProtobuf{}, onConnected)
}
