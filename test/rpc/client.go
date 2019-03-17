package main

import (
	"github.com/temprory/log"
	"github.com/temprory/net"
	"time"
)

var (
	addr = "127.0.0.1:8888"
)

type HelloRequest struct {
	Name string
}

type HelloReply struct {
	Message string
}

func main() {
	//client, err := net.NewRpcClient(addr, nil, nil, nil)
	poolSize := 10
	client, err := net.NewRpcClientPool(addr, nil, nil, poolSize, nil)
	if err != nil {
		log.Debug("NewReqClient Error: ", err)
		return
	}

	req := &HelloRequest{Name: "temprory"}
	rsp := &HelloReply{}

	err = client.CallMethodWithTimeout("Hello", req, rsp, time.Second*3)
	if err != nil {
		log.Error("Hello failed: %v", err)
		return
	}

	log.Info("Hello: %v", req.Name == rsp.Message)
}
