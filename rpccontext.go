package net

import (
	"bytes"
	"encoding/gob"
	"errors"
	"github.com/golang/protobuf/proto"
	"github.com/json-iterator/go"
	"github.com/vmihailenco/msgpack"
)

var (
	ErrInvalidBody = errors.New("invalid body")

	json = jsoniter.ConfigCompatibleWithStandardLibrary
)

type RpcContext struct {
	method  string
	client  ITcpClient
	message IMessage
}

func (ctx *RpcContext) Client() ITcpClient {
	return ctx.client
}

func (ctx *RpcContext) Cmd() uint32 {
	return ctx.message.Cmd()
}

func (ctx *RpcContext) Body() []byte {
	return ctx.message.Body()
}

func (ctx *RpcContext) Method() string {
	return ctx.method
}

func (ctx *RpcContext) WriteData(data []byte) error {

	return ctx.client.pushDataSync(NewRpcMessage(ctx.message.Cmd(), ctx.message.RpcSeq(), data).data)
}

func (ctx *RpcContext) WriteMsg(msg IMessage) error {
	if ctx.message != msg {
		msg.SetRpcSeq(ctx.message.RpcSeq())
	}
	return ctx.client.pushDataSync(msg.Data())
}

func (ctx *RpcContext) Bind(v interface{}) error {
	return DefaultRpcCodec.Unmarshal(ctx.Body(), v)
}

func (ctx *RpcContext) Write(v interface{}) error {
	data, err := DefaultRpcCodec.Marshal(v)
	if err != nil {
		return err
	}
	return ctx.client.pushDataSync(NewRpcMessage(ctx.message.Cmd(), ctx.message.RpcSeq(), data).data)
}

func (ctx *RpcContext) BindJson(v interface{}) error {
	return json.Unmarshal(ctx.Body(), v)
}

func (ctx *RpcContext) WriteJson(v interface{}) error {
	data, err := json.Marshal(v)
	if err != nil {
		return err
	}
	return ctx.client.pushDataSync(NewRpcMessage(ctx.message.Cmd(), ctx.message.RpcSeq(), data).data)
}

func (ctx *RpcContext) BindGob(v interface{}) error {
	return gob.NewDecoder(bytes.NewBuffer(ctx.Body())).Decode(v)
}

func (ctx *RpcContext) WriteGob(v interface{}) error {
	buffer := &bytes.Buffer{}
	err := gob.NewEncoder(buffer).Encode(v)
	if err != nil {
		return err
	}
	return ctx.client.pushDataSync(NewRpcMessage(ctx.message.Cmd(), ctx.message.RpcSeq(), buffer.Bytes()).data)
}

func (ctx *RpcContext) BindMsgpack(v interface{}) error {
	return msgpack.Unmarshal(ctx.Body(), v)
}

func (ctx *RpcContext) WriteMsgpack(v interface{}) error {
	data, err := msgpack.Marshal(v)
	if err != nil {
		return err
	}
	return ctx.client.pushDataSync(NewRpcMessage(ctx.message.Cmd(), ctx.message.RpcSeq(), data).data)
}

func (ctx *RpcContext) BindProtobuf(v proto.Message) error {
	return proto.Unmarshal(ctx.Body(), v)
}

func (ctx *RpcContext) WriteProtobuf(v proto.Message) error {
	data, err := proto.Marshal(v)
	if err != nil {
		return err
	}
	return ctx.client.pushDataSync(NewRpcMessage(ctx.message.Cmd(), ctx.message.RpcSeq(), data).data)
}

func (ctx *RpcContext) Error(errText string) {
	ctx.client.pushDataSync(NewRpcMessage(CmdRpcError, ctx.message.RpcSeq(), []byte(errText)).data)
}
