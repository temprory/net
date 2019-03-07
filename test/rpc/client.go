package main

import (
	//"encoding/gob"
	// "bytes"
	// "encoding/gob"
	"fmt"
	"github.com/json-iterator/go"
	"github.com/temprory/log"
	"github.com/temprory/net"
	"github.com/temprory/net/test/rpc/pb"
	//"runtime"
	"sync"
	"sync/atomic"
	"time"
)

var (
	qps  = int64(0)
	addr = "localhost:8888"
	json = jsoniter.ConfigCompatibleWithStandardLibrary
)

var (
	CMD_GOB_RPC      = uint32(1)
	CMD_JSON_RPC     = uint32(2)
	CMD_MSGPACK_RPC  = uint32(3)
	CMD_PROTOBUF_RPC = uint32(4)

	wg = sync.WaitGroup{}

	data        = []byte{}
	clientNum   = int64(20 / 10)
	loopNum     = int64(10000)
	totalReqNum = int64(0)
)

type HelloRequest struct {
	Name string
}

// The response message containing the greetings
type HelloReply struct {
	Message string
}

// type RpcClient struct {
// 	c net.IRpcClient
// }

// func (client *RpcClient) CallJsonWithTimeout(cmd uint32, req interface{}, rsp interface{}, timeout time.Duration) error {
// 	data, err := json.Marshal(req)
// 	if err != nil {
// 		log.Debug("rpc failed: %v", err)
// 		return err
// 	}
// 	rspdata, err := client.c.CallWithTimeout(cmd, data, timeout)
// 	if err != nil {
// 		log.Debug("rpc failed: %v", err)
// 		return err
// 	}
// 	if rsp != nil {
// 		err = json.Unmarshal(rspdata, rsp)
// 		if err != nil {
// 			log.Debug("rpc failed: %v", err)
// 		}
// 	}
// 	return err
// }

// func (client *RpcClient) CallGobWithTimeout(cmd uint32, req interface{}, rsp interface{}, timeout time.Duration) error {
// 	buffer := &bytes.Buffer{}
// 	err := gob.NewEncoder(buffer).Encode(req)
// 	if err != nil {
// 		log.Debug("rpc failed: %v", err)
// 		return err
// 	}
// 	rspdata, err := client.c.CallWithTimeout(cmd, buffer.Bytes(), timeout)
// 	if err != nil {
// 		log.Debug("rpc failed: %v", err)
// 		return err
// 	}
// 	if rsp != nil {
// 		gob.NewDecoder(bytes.NewBuffer(rspdata)).Decode(rsp)
// 	}
// 	return err
// }

func startGobRpcCmdClient() {
	defer wg.Done()

	client, err := net.NewGobRpcClient(addr, nil, nil)
	if err != nil {
		log.Debug("NewReqClient Error: ", err)
	}
	atomic.AddInt64(&totalReqNum, loopNum)
	for i := int64(0); i < loopNum; i++ {
		req := &HelloRequest{Name: fmt.Sprintf("hello_%d", i)}
		rsp := &HelloReply{}
		err := client.CallCmdWithTimeout(CMD_GOB_RPC, req, rsp, time.Second*3)
		if err != nil {
			log.Debug("gobrpc cmd failed: %v", err)
			continue
		}
		if rsp.Message != req.Name {
			log.Debug("gobrpc cmd failed: %v, %v, %v", err, rsp.Message, req.Name)
		}
		atomic.AddInt64(&qps, 1)
	}
}

func startGobRpcMethodClient() {
	defer wg.Done()

	client, err := net.NewGobRpcClient(addr, nil, nil)
	if err != nil {
		log.Debug("NewReqClient Error: ", err)
	}
	atomic.AddInt64(&totalReqNum, loopNum)
	for i := int64(0); i < loopNum; i++ {
		req := &HelloRequest{Name: fmt.Sprintf("hello_%d", i)}
		rsp := &HelloReply{}
		err := client.CallMethodWithTimeout("GobRpc.Hello", req, rsp, time.Second*3)
		if err != nil {
			log.Debug("gobrpc method failed: %v", err)
			continue
		}
		if rsp.Message != req.Name {
			log.Debug("gobrpc method failed: %v, %v, %v", err, rsp.Message, req.Name)
		}
		atomic.AddInt64(&qps, 1)
	}
}

func startJsonRpcCmdClient() {
	defer wg.Done()

	client, err := net.NewJsonRpcClient(addr, nil, nil)
	if err != nil {
		log.Debug("NewReqClient Error: ", err)
	}
	atomic.AddInt64(&totalReqNum, loopNum)
	for i := int64(0); i < loopNum; i++ {
		req := &HelloRequest{Name: fmt.Sprintf("hello_%d", i)}
		rsp := &HelloReply{}
		err := client.CallCmdWithTimeout(CMD_JSON_RPC, req, rsp, time.Second*3)
		if err != nil {
			log.Debug("jsonrpc cmd failed: %v", err)
			continue
		}
		if rsp.Message != req.Name {
			log.Debug("jsonrpc cmd failed: %v", err)
		}
		atomic.AddInt64(&qps, 1)
	}
}

func startJsonRpcMethodClient() {
	defer wg.Done()

	client, err := net.NewJsonRpcClient(addr, nil, nil)
	if err != nil {
		log.Debug("NewReqClient Error: ", err)
	}
	atomic.AddInt64(&totalReqNum, loopNum)
	for i := int64(0); i < loopNum; i++ {
		req := &HelloRequest{Name: fmt.Sprintf("hello_%d", i)}
		rsp := &HelloReply{}
		err := client.CallMethodWithTimeout("JsonRpc.Hello", req, rsp, time.Second*3)
		if err != nil {
			log.Debug("jsonrpc method failed: %v", err)
			continue
		}
		if rsp.Message != req.Name {
			log.Debug("jsonrpc method failed: %v", err)
		}
		atomic.AddInt64(&qps, 1)
	}
}

func startMsgpackRpcCmdClient() {
	defer wg.Done()

	client, err := net.NewMsgpackRpcClient(addr, nil, nil)
	if err != nil {
		log.Debug("NewReqClient Error: ", err)
	}
	atomic.AddInt64(&totalReqNum, loopNum)
	for i := int64(0); i < loopNum; i++ {
		req := &HelloRequest{Name: fmt.Sprintf("hello_%d", i)}
		rsp := &HelloReply{}
		err := client.CallCmdWithTimeout(CMD_MSGPACK_RPC, req, rsp, time.Second*3)
		if err != nil {
			log.Debug("msgpackrpc cmd failed: %v", err)
			continue
		}
		if rsp.Message != req.Name {
			log.Debug("msgpackrpc cmd failed: %v", err)
		}
		atomic.AddInt64(&qps, 1)
	}
}

func startMsgpackRpcMethodClient() {
	defer wg.Done()

	client, err := net.NewMsgpackRpcClient(addr, nil, nil)
	if err != nil {
		log.Debug("NewReqClient Error: ", err)
	}
	atomic.AddInt64(&totalReqNum, loopNum)
	for i := int64(0); i < loopNum; i++ {
		req := &HelloRequest{Name: fmt.Sprintf("hello_%d", i)}
		rsp := &HelloReply{}
		err := client.CallMethodWithTimeout("MsgpackRpc.Hello", req, rsp, time.Second*3)
		if err != nil {
			log.Debug("msgpackrpc mechod failed: %v", err)
			continue
		}
		if rsp.Message != req.Name {
			log.Debug("msgpackrpc mechod failed: %v", err)
		}
		atomic.AddInt64(&qps, 1)
	}
}

func startProtobufRpcCmdClient() {
	defer wg.Done()

	client, err := net.NewProtobufRpcClient(addr, nil, nil)
	if err != nil {
		log.Debug("NewReqClient Error: ", err)
	}
	atomic.AddInt64(&totalReqNum, loopNum)
	for i := int64(0); i < loopNum; i++ {
		req := &pb.HelloRequest{Name: fmt.Sprintf("hello_%d", i)}
		rsp := &pb.HelloReply{}
		err := client.CallCmdWithTimeout(CMD_PROTOBUF_RPC, req, rsp, time.Second*3)
		if err != nil {
			log.Debug("protobufrpc cmd failed: %v", err)
			continue
		}
		if rsp.Message != req.Name {
			log.Debug("protobufrpc cmd failed: %v", err)
		}
		atomic.AddInt64(&qps, 1)
	}
}

func startProtobufRpcMethodClient() {
	defer wg.Done()

	client, err := net.NewProtobufRpcClient(addr, nil, nil)
	if err != nil {
		log.Debug("NewReqClient Error: %v", err)
	}
	atomic.AddInt64(&totalReqNum, loopNum)
	for i := int64(0); i < loopNum; i++ {
		req := &pb.HelloRequest{Name: fmt.Sprintf("hello_%d", i)}
		rsp := &pb.HelloReply{}
		err := client.CallMethodWithTimeout("ProtobufRpc.Hello", req, rsp, time.Second*3)
		if err != nil {
			log.Debug("protobufrpc mechod failed: %v", err)
			continue
		}
		if rsp.Message != req.Name {
			log.Debug("protobufrpc mechod failed: %v", err)
		}
		atomic.AddInt64(&qps, 1)
	}
}

func startRpcWithCodecMethodClient() {
	defer wg.Done()

	client, err := net.NewRpcClient(addr, nil, json, nil)
	if err != nil {
		log.Debug("NewReqClient Error: ", err)
	}
	atomic.AddInt64(&totalReqNum, loopNum)
	for i := int64(0); i < loopNum; i++ {
		req := &HelloRequest{Name: fmt.Sprintf("hello_%d", i)}
		rsp := &HelloReply{}
		err := client.CallMethodWithTimeout("JsonRpc.Hello", req, rsp, time.Second*3)
		if err != nil {
			log.Debug("jsonrpc method failed: %v", err)
			continue
		}
		if rsp.Message != req.Name {
			log.Debug("jsonrpc method failed: %v", err)
		}
		atomic.AddInt64(&qps, 1)
	}
}

func startRpcWithDefaultCodecMethodClient() {
	defer wg.Done()

	client, err := net.NewRpcClient(addr, nil, nil, nil)
	if err != nil {
		log.Debug("NewReqClient Error: ", err)
	}
	atomic.AddInt64(&totalReqNum, loopNum)
	for i := int64(0); i < loopNum; i++ {
		req := &HelloRequest{Name: fmt.Sprintf("hello_%d", i)}
		rsp := &HelloReply{}
		err := client.CallMethodWithTimeout("MsgpackRpc.Hello", req, rsp, time.Second*3)
		if err != nil {
			log.Debug("default codec msgpack method failed: %v", err)
			continue
		}
		if rsp.Message != req.Name {
			log.Debug("default codec msgpack method failed: %v", err)
		}
		atomic.AddInt64(&qps, 1)
	}
}

func main() {
	t0 := time.Now()
	for i := int64(0); i < clientNum; i++ {
		wg.Add(1)
		go startGobRpcCmdClient()
		wg.Add(1)
		go startGobRpcMethodClient()

		wg.Add(1)
		go startJsonRpcCmdClient()
		wg.Add(1)
		go startJsonRpcMethodClient()

		wg.Add(1)
		go startMsgpackRpcCmdClient()
		wg.Add(1)
		go startMsgpackRpcMethodClient()

		wg.Add(1)
		go startProtobufRpcCmdClient()
		wg.Add(1)
		go startProtobufRpcMethodClient()

		wg.Add(1)
		go startRpcWithCodecMethodClient()
		wg.Add(1)
		go startRpcWithDefaultCodecMethodClient()
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
	//<-make(chan int)
}
