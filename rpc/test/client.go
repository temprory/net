package main

import (
	"fmt"
	"rpc"
	"time"
)

type HelloReq struct {
	Message string
	Time    time.Time
}

type HelloRsp struct {
	Message string
	Time    time.Time
}

func main() {
	//client := rpc.NewClient("127.0.0.1:8888")
	client := rpc.NewClientPool("127.0.0.1:8888", 10)

	for i := 0; i < 10; i++ {
		var req = HelloReq{Message: fmt.Sprintf("hello %v", i), Time: time.Now()}
		var rsp HelloRsp
		err := client.Call("echo", &req, &rsp, time.Second)
		if err != nil {
			fmt.Println("call echo failed:", err)
			continue
		}
		fmt.Println("client call echo:", req.Message, req.Time.UnixNano(), rsp.Time.UnixNano())

		retstr := ""
		err = client.Call("empty", nil, &retstr, time.Second)
		if err != nil {
			fmt.Println("call empty failed:", err)
			continue
		}
		fmt.Println("client call empty:", i, retstr)

		time.Sleep(time.Second / 20)
	}
}
