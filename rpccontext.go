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
	Client ITcpClient
	Msg    IMessage
}

func (ctx *RpcContext) Cmd() uint32 {
	return ctx.Msg.Cmd()
}

func (ctx *RpcContext) Body() []byte {
	return ctx.Msg.Body()
}

func (ctx *RpcContext) Write(data []byte) error {
	return ctx.Client.SendMsg(NewRpcMessage(ctx.Msg.Cmd(), ctx.Msg.RpcSeq(), data))
}

func (ctx *RpcContext) WriteMsg(msg IMessage) error {
	if ctx.Msg != msg {
		msg.SetRpcSeq(ctx.Msg.RpcSeq())
	}
	return ctx.Client.SendMsg(msg)
}

func (ctx *RpcContext) BindJson(v interface{}) error {
	return json.Unmarshal(ctx.Body(), v)
}

func (ctx *RpcContext) WriteJson(v interface{}) error {
	data, err := json.Marshal(v)
	if err != nil {
		return err
	}
	return ctx.Client.SendMsg(NewRpcMessage(ctx.Msg.Cmd(), ctx.Msg.RpcSeq(), data))
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
	return ctx.Client.SendMsg(NewRpcMessage(ctx.Msg.Cmd(), ctx.Msg.RpcSeq(), buffer.Bytes()))
}

func (ctx *RpcContext) BindMsgPack(v interface{}) error {
	return msgpack.Unmarshal(ctx.Body(), v)
}

func (ctx *RpcContext) WriteMsgPack(v interface{}) error {
	data, err := msgpack.Marshal(v)
	if err != nil {
		return err
	}
	return ctx.Client.SendMsg(NewRpcMessage(ctx.Msg.Cmd(), ctx.Msg.RpcSeq(), data))
}

func (ctx *RpcContext) BindProtobuf(v proto.Message) error {
	return proto.Unmarshal(ctx.Body(), v)
}

func (ctx *RpcContext) WriteProtobuf(v proto.Message) error {
	data, err := proto.Marshal(v)
	if err != nil {
		return err
	}
	return ctx.Client.SendMsg(NewRpcMessage(ctx.Msg.Cmd(), ctx.Msg.RpcSeq(), data))
}
