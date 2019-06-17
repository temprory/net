package main

import (
	"fmt"
	"github.com/temprory/log"
	"github.com/temprory/net"
	"sync/atomic"
	"time"
)

var (
	addr = "127.0.0.1:8888"

	qpsEcho    = int64(0)
	qpsCliCall = int64(0)
	qpsSvrCall = int64(0)

	CMD_ECHO     = uint32(6666)
	CMD_SVR_CALL = uint32(8888)
)

type HelloRequest struct {
	Name string
}

type HelloReply struct {
	Message string
}

func onEcho(client *net.TcpClient, msg *net.Message) {
	//log.Info("onEcho: %v", string(msg.Body()))
	atomic.AddInt64(&qpsEcho, 1)
	time.Sleep(time.Second / 10)
	client.SendMsg(msg)
}

func onSvrCall(client *net.TcpClient, msg *net.Message) {
	//log.Info("onSvrCall: %v", string(msg.Body()))
	atomic.AddInt64(&qpsSvrCall, 1)
}

func main() {
	//qps log
	go func() {
		for {
			time.Sleep(time.Second)
			qE, qC, qS := atomic.SwapInt64(&qpsEcho, 0), atomic.SwapInt64(&qpsCliCall, 0), atomic.SwapInt64(&qpsSvrCall, 0)
			fmt.Printf("qpsTotal: %v, qpsEcho: %v, qpsCliCall: %v, qpsSvrCall: %v\n", qE+qC+qS, qE, qC, qS)
		}
	}()

	poolSize := 5
	engine := net.NewTcpEngine()
	engine.Handle(CMD_ECHO, onEcho)
	engine.Handle(CMD_SVR_CALL, onSvrCall)

	pool, err := net.NewRpcClientPool(addr, engine, nil, poolSize, func(c *net.TcpClient) {
		//client send cmd msg to server
		c.SendMsg(net.NewMessage(CMD_ECHO, []byte(fmt.Sprintf("hello_%v", 2))))

	})
	if err != nil {
		log.Error("NewReqClient Error: %v", err)
		return
	}

	//client call server method
	for {
		req := &HelloRequest{Name: "temprory"}
		rsp := &HelloReply{}

		err = pool.Call("Hello", req, rsp, time.Second*3)
		if err != nil {
			log.Error("Hello failed: %v", err)
			//return
		}
		atomic.AddInt64(&qpsCliCall, 1)
		time.Sleep(time.Second / 10)
	}
}
