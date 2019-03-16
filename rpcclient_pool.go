package net

import (
	"errors"
	"time"
)

type RpcClientPool struct {
	codec   IRpcCodec
	clients chan *RpcClient
}

func (pool *RpcClientPool) Codec() IRpcCodec {
	return pool.codec
}

func (pool *RpcClientPool) CallCmd(cmd uint32, req interface{}, rsp interface{}) error {
	client := <-pool.clients
	err := client.CallCmd(cmd, req, rsp)
	pool.clients <- client
	return err
}

func (pool *RpcClientPool) CallCmdWithTimeout(cmd uint32, req interface{}, rsp interface{}, timeout time.Duration) error {
	select {
	case client := <-pool.clients:
		err := client.CallCmdWithTimeout(cmd, req, rsp, timeout)
		pool.clients <- client
		return err
	case <-time.After(timeout):
	}

	return ErrRpcCallTimeout
}

func (pool *RpcClientPool) CallMethod(method string, req interface{}, rsp interface{}) error {
	client := <-pool.clients
	err := client.CallMethod(method, req, rsp)
	pool.clients <- client
	return err
}

func (pool *RpcClientPool) CallMethodWithTimeout(method string, req interface{}, rsp interface{}, timeout time.Duration) error {
	select {
	case client := <-pool.clients:
		err := client.CallMethodWithTimeout(method, req, rsp, timeout)
		pool.clients <- client
		return err
	case <-time.After(timeout):
	}
	return ErrRpcCallTimeout
}

func NewRpcClientPool(addr string, engine ITcpEngin, codec IRpcCodec, poolSize int, onConnected func(ITcpClient)) (*RpcClientPool, error) {
	if engine == nil {
		engine = NewTcpEngine()
	}
	engine.SetSendQueueSize(_conf_sock_rpc_send_q_size)
	engine.SetSockRecvBlockTime(_conf_sock_rpc_recv_block_time)

	clients := map[ITcpClient]*RpcClient{}
	engine.HandleOnMessage(func(c ITcpClient, msg IMessage) {
		//if engine.running {
		rpcclient := clients[c]
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
				logDebug("no rpcsession waiting for rpc response, cmd %X, ip: %v", msg.Cmd(), c.Ip())
			}
		}
		// } else {
		// 	logDebug("engine is not running, ignore rpc cmd %X, ip: %v", msg.Cmd(), client.Ip())
		// }
	})

	if codec == nil {
		codec = DefaultRpcCodec
		logDebug("use default rpc codec: %v", defaultRpcCodecType)
	}

	pool := &RpcClientPool{
		codec:   codec,
		clients: make(chan *RpcClient, poolSize*100),
	}

	for i := 0; i < poolSize; i++ {
		client, err := NewTcpClient(addr, engine, nil, true, onConnected)
		if err != nil {
			return nil, err
		}
		safeGo(func() {
			client.Keepalive(_conf_sock_keepalive_time)
		})

		rpcclient := &RpcClient{client, map[int64]*rpcsession{}, pool.codec}
		rpcclient.OnClose("-", func(ITcpClient) {
			rpcclient.Lock()
			defer rpcclient.Unlock()
			for _, session := range rpcclient.sessionMap {
				close(session.done)
			}
			rpcclient.sessionMap = map[int64]*rpcsession{}
		})

		clients[client] = rpcclient
	}

	for i := 0; i < 100; i++ {
		for _, v := range clients {
			pool.clients <- v
		}
	}

	return pool, nil
}
