package main

import (
	"github.com/json-iterator/go"
	"github.com/temprory/log"
	"github.com/temprory/net"
	"github.com/temprory/net/test/rpc/pb"
	"time"
)

var (
	json = jsoniter.ConfigCompatibleWithStandardLibrary
)

const (
	CMD_GOB_RPC      = uint32(1)
	CMD_JSON_RPC     = uint32(2)
	CMD_MSGPACK_RPC  = uint32(3)
	CMD_PROTOBUF_RPC = uint32(4)
)

type HelloRequest struct {
	Name string
}

// The response message containing the greetings
type HelloReply struct {
	Message string
}

func onGobRpc(ctx *net.RpcContext) {
	req := &HelloRequest{}
	err := ctx.BindGob(req)
	if err != nil {
		ctx.Error("invalid data")
		log.Error("onGobRpc failed: %v", err)
		return
	}

	err = ctx.WriteGob(&HelloReply{Message: req.Name})
	if err != nil {
		log.Error("onGobRpc failed: %v", err)
	}
}

func onJsonRpc(ctx *net.RpcContext) {
	req := &HelloRequest{}
	err := ctx.BindJson(req)
	if err != nil {
		ctx.Error("invalid data")
		log.Error("onJsonRpc failed: %v", err)
		return
	}

	err = ctx.WriteJson(&HelloReply{Message: req.Name})
	if err != nil {
		log.Error("onJsonRpc failed: %v", err)
	}
}

func onMsgpackRpc(ctx *net.RpcContext) {
	req := &HelloRequest{}
	err := ctx.BindMsgpack(req)
	if err != nil {
		ctx.Error("invalid data")
		log.Error("onMsgpackRpc failed: %v", err)
		return
	}
	//log.Info("-- onMsgpackRpc: %v", req.Name)
	err = ctx.WriteMsgpack(&HelloReply{Message: req.Name})
	if err != nil {

		log.Error("onMsgpackRpc failed: %v", err)
	}
}

func onProtobufRpc(ctx *net.RpcContext) {
	req := &pb.HelloRequest{}
	err := ctx.BindProtobuf(req)
	if err != nil {
		ctx.Error("invalid data")
		log.Error("onProtobufRpc failed: %v", err)
		return
	}
	//log.Info("-- onProtobufRpc: %v", req.Name)
	err = ctx.WriteProtobuf(&pb.HelloReply{Message: req.Name})
	if err != nil {
		log.Error("onProtobufRpc failed: %v", err)
	}
}

func main() {
	server := net.NewTcpServer("rpc")
	server.SetMaxConcurrent(1000)
	server.HandleRpcCmd(CMD_GOB_RPC, onGobRpc, false)
	server.HandleRpcCmd(CMD_JSON_RPC, onJsonRpc, false)
	server.HandleRpcCmd(CMD_MSGPACK_RPC, onMsgpackRpc, false)
	server.HandleRpcCmd(CMD_PROTOBUF_RPC, onProtobufRpc, false)
	server.HandleRpcMethod("GobRpc.Hello", onGobRpc, false)
	server.HandleRpcMethod("JsonRpc.Hello", onJsonRpc, false)
	server.HandleRpcMethod("MsgpackRpc.Hello", onMsgpackRpc, false)
	server.HandleRpcMethod("ProtobufRpc.Hello", onProtobufRpc, false)

	server.Serve("localhost:8888", time.Second*5)
}
