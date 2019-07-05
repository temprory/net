package main

import (
	"fmt"
	"rpc"
	"time"
)

type HelloReq struct {
	IMessage string
	Time     time.Time
}

type HelloRsp struct {
	IMessage string
	Time     time.Time
}

func echo(ctx *rpc.Context) {
	var req HelloReq
	var rsp HelloRsp
	err := ctx.Bind(&req)
	if err != nil {
		fmt.Println("bind failed:", err)
		ctx.Error("invalid data")
		return
	}
	rsp.IMessage = req.IMessage

	rsp.Time = time.Now()
	ctx.Write(&rsp)

	fmt.Println("server echo:", req.IMessage, req.Time.UnixNano(), rsp.Time.UnixNano())
}

func empty(ctx *rpc.Context) {
	ctx.Write(time.Now().Format("2006-01-02 15:04:03.000"))
}

func main() {
	svr := &rpc.Server{}
	svr.Handle("echo", echo)
	svr.Handle("empty", empty)
	go func() {
		time.Sleep(time.Second * 20)
		svr.Shutdown()
	}()
	svr.ListenAndServe(":8888")
}
