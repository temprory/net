package main

import (
	"fmt"
	"github.com/temprory/log"
	"github.com/temprory/net"
	"sync"
	"sync/atomic"
	"time"
)

var (
	qps  = int64(0)
	addr = "localhost:8888"
)

var (
	wg = sync.WaitGroup{}

	data        = []byte{}
	clientNum   = int64(20 / 1)
	loopNum     = int64(1000000)
	totalReqNum = int64(0)
)

type HelloRequest struct {
	Name string
}

// The response message containing the greetings
type HelloReply struct {
	IMessage string
}

func oneClient() {
	defer wg.Done()

	client, err := net.NewRpcClient(addr, nil, nil, nil)
	if err != nil {
		log.Debug("NewReqClient Error: ", err)
	}
	atomic.AddInt64(&totalReqNum, loopNum)
	for i := int64(0); i < loopNum; i++ {
		req := &HelloRequest{Name: fmt.Sprintf("hello_%d", i)}
		rsp := &HelloReply{}
		err := client.Call("JsonRpc.Hello", req, rsp, time.Second*3)
		if err != nil {
			log.Debug("default codec msgpack method failed: %v", err)
			continue
		}
		if rsp.IMessage != req.Name {
			log.Debug("default codec msgpack method failed: %v", err)
		}
		atomic.AddInt64(&qps, 1)
	}
}

func main() {
	t0 := time.Now()
	for i := int64(0); i < clientNum; i++ {
		wg.Add(1)
		go oneClient()
	}
	go func() {
		for {
			time.Sleep(time.Second)
			fmt.Println("qps: ", atomic.SwapInt64(&qps, 0))
		}
	}()
	wg.Wait()
	seconds := time.Since(t0).Seconds()
	log.Debug("total used: %v, request: %d, %d / s", seconds, totalReqNum, int(float64(totalReqNum)/seconds))
}
