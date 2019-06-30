## Echo Test

- server

```sh
cd rpc

go run ./test/server.go
```

- client

```sh
cd rpc

go run ./test/client.go
```

## 代码

- server

```golang
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

func echo(ctx *rpc.Context) {
	var req HelloReq
	var rsp HelloRsp
	err := ctx.Bind(&req)
	if err != nil {
		fmt.Println("bind failed:", err)
		ctx.Error("invalid data")
		return
	}
	rsp.Message = req.Message

	time.Sleep(10)
	rsp.Time = time.Now()
	ctx.Write(&rsp)

	fmt.Println("server echo:", req.Message, req.Time.UnixNano(), rsp.Time.UnixNano())
}

func main() {
	svr := &rpc.Server{}
	svr.Handle("echo", echo)
	svr.ListenAndServe(":8888")
}

```

- client

```golang
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
	client := rpc.NewClient("127.0.0.1:8888")

	for i := 0; i < 10; i++ {
		var req = HelloReq{Message: fmt.Sprintf("hello %v", i), Time: time.Now()}
		var rsp HelloRsp
		err := client.Call("echo", &req, &rsp, time.Second)
		if err != nil {
			fmt.Println("call failed:", err)
			continue
		}
		fmt.Println("client echo:", req.Message, req.Time.UnixNano(), rsp.Time.UnixNano())
	}
}

```