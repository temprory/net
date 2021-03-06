package net

import (
	"errors"
	"sync/atomic"
	"time"
)

type RpcClientPool struct {
	idx     int64
	size    uint64
	codec   ICodec
	clients []*RpcClient
}

func (pool *RpcClientPool) Codec() ICodec {
	return pool.codec
}

// func (pool *RpcClientPool) CallCmd(cmd uint32, req interface{}, rsp interface{}) error {
// 	idx := atomic.AddInt64(&pool.idx, 1)
// 	client := pool.clients[uint64(idx)%pool.size]
// 	err := client.CallCmd(cmd, req, rsp)
// 	return err
// }

// func (pool *RpcClientPool) CallCmdWithTimeout(cmd uint32, req interface{}, rsp interface{}, timeout time.Duration) error {
// 	idx := atomic.AddInt64(&pool.idx, 1)
// 	client := pool.clients[uint64(idx)%pool.size]
// 	err := client.CallCmdWithTimeout(cmd, req, rsp, timeout)
// 	return err
// }

// func (pool *RpcClientPool) CallMethod(method string, req interface{}, rsp interface{}) error {
// 	idx := atomic.AddInt64(&pool.idx, 1)
// 	client := pool.clients[uint64(idx)%pool.size]
// 	err := client.CallMethod(method, req, rsp)
// 	return err
// }

// func (pool *RpcClientPool) CallMethodWithTimeout(method string, req interface{}, rsp interface{}, timeout time.Duration) error {
// 	idx := atomic.AddInt64(&pool.idx, 1)
// 	client := pool.clients[uint64(idx)%pool.size]
// 	err := client.CallMethodWithTimeout(method, req, rsp, timeout)
// 	return err
// }

func (pool *RpcClientPool) Client() *RpcClient {
	idx := atomic.AddInt64(&pool.idx, 1)
	return pool.clients[uint64(idx)%pool.size]
}

func (pool *RpcClientPool) Call(method string, req interface{}, rsp interface{}, timeout time.Duration) error {
	idx := atomic.AddInt64(&pool.idx, 1)
	client := pool.clients[uint64(idx)%pool.size]
	err := client.Call(method, req, rsp, timeout)
	return err
}

func NewRpcClientPool(addr string, engine *TcpEngin, codec ICodec, poolSize int, onConnected func(*TcpClient)) (*RpcClientPool, error) {
	if engine == nil {
		engine = NewTcpEngine()
		engine.SetSendQueueSize(DefaultSockRpcSendQSize)
		engine.SetSockRecvBlockTime(DefaultSockRpcRecvBlockTime)
	}

	clients := map[*TcpClient]*RpcClient{}
	engine.HandleOnMessage(func(c *TcpClient, msg IMessage) {
		//if engine.running {
		switch msg.Cmd() {
		case CmdPing:
		case CmdRpcMethod:
			rpcclient := clients[c]
			rpcclient.Lock()
			session, ok := rpcclient.sessionMap[msg.RpcSeq()]
			rpcclient.Unlock()
			if ok {
				session.done <- &RpcIMessage{msg, nil}
			} else {
				logDebug("no rpcsession waiting for rpc response, cmd %X, ip: %v", msg.Cmd(), c.Ip())
			}
		case CmdRpcError:
			rpcclient := clients[c]
			rpcclient.Lock()
			session, ok := rpcclient.sessionMap[msg.RpcSeq()]
			rpcclient.Unlock()
			if ok {
				session.done <- &RpcIMessage{msg, errors.New(string(msg.Body()))}
			} else {
				logDebug("no rpcsession waiting for rpc response, cmd %X, ip: %v", msg.Cmd(), c.Ip())
			}
		default:
			if handler, ok := engine.handlerMap[msg.Cmd()]; ok {
				engine.Add(1)
				defer engine.Done()
				defer handlePanic()
				handler(c, msg)
			} else {
				logDebug("no handler for cmd %v", msg.Cmd())
			}
		}
		// } else {
		// 	logDebug("engine is not running, ignore rpc cmd %X, ip: %v", msg.Cmd(), client.Ip())
		// }
	})

	if codec == nil {
		codec = DefaultCodec
		logDebug("use default rpc codec: %v", DefaultRpcCodecType)
	}

	pool := &RpcClientPool{
		size:    uint64(poolSize),
		codec:   codec,
		clients: make([]*RpcClient, poolSize),
	}

	for i := 0; i < poolSize; i++ {
		client, err := NewTcpClient(addr, engine, NewCipherGzip(DefaultThreshold), true, onConnected)
		if err != nil {
			return nil, err
		}
		safeGo(func() {
			client.Keepalive(DefaultSockKeepaliveTime)
		})

		rpcclient := &RpcClient{client, map[int64]*rpcsession{}, pool.codec}
		rpcclient.OnClose("-", func(*TcpClient) {
			rpcclient.Lock()
			defer rpcclient.Unlock()
			for _, session := range rpcclient.sessionMap {
				close(session.done)
			}
			rpcclient.sessionMap = map[int64]*rpcsession{}
		})

		clients[client] = rpcclient
		pool.clients[i] = rpcclient
	}

	return pool, nil
}
