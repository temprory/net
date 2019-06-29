package main

import (
	"context"
	"flag"
	"fmt"
	example "github.com/rpcx-ecosystem/rpcx-examples3"
	"github.com/smallnest/rpcx/client"
	"github.com/temprory/log"
	"sync"
	"sync/atomic"
	"time"
)

var (
	qps = int64(0)

	wg = sync.WaitGroup{}

	data        = []byte{}
	clientNum   = int64(16)
	loopNum     = int64(500000)
	totalReqNum = int64(0)
)
var (
	addr2 = flag.String("addr", "localhost:8972", "server address")
)

func startRpcClient() {
	defer wg.Done()
	d := client.NewPeer2PeerDiscovery("tcp@"+*addr2, "")
	xclient := client.NewXClient("Arith", client.Failtry, client.RandomSelect, d, client.DefaultOption)
	defer xclient.Close()

	atomic.AddInt64(&totalReqNum, loopNum)
	for i := int64(0); i < loopNum; i++ {
		req := &example.HelloRequest{Name: fmt.Sprintf("hello_%d", i)}
		rsp := &example.HelloReply{}

		call, err := xclient.Go(context.Background(), "Say", req, rsp, nil)

		if err != nil {
			log.Debug("protobufrpc mechod failed: %v", err)
		}

		replyCall := <-call.Done
		if replyCall.Error != nil {
			log.Error("failed to call: %v", replyCall.Error)
		} else {
			if rsp.Message != req.Name {
				log.Debug("protobufrpc mechod failed: %v", err)
			}
		}
		atomic.AddInt64(&qps, 1)
	}
}

func main() {
	t0 := time.Now()
	for i := int64(0); i < clientNum; i++ {
		wg.Add(1)
		go startRpcClient()
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
