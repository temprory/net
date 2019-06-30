package rpc

import (
	"fmt"
	"github.com/valyala/fastrpc/tlv"
)

type Context struct {
	ctx  *tlv.RequestCtx
	data []byte
}

func (ctx *Context) Bind(v interface{}) error {
	return DefaultRpcCodec.Unmarshal(ctx.data, v)
}

func (ctx *Context) Write(v interface{}) (int, error) {
	if v == nil {
		return ctx.ctx.Write([]byte{cmd_msg})
	}
	if data, ok := v.([]byte); ok {
		return ctx.ctx.Write(append([]byte{cmd_msg}, data...))
	}
	if data, ok := v.(string); ok {
		return ctx.ctx.Write(append([]byte{cmd_msg}, []byte(data)...))
	}
	data, err := DefaultRpcCodec.Marshal(v)
	if err == nil {
		return ctx.ctx.Write(append([]byte{cmd_msg}, data...))
	}
	return 0, err
}

func (ctx *Context) Error(err interface{}) (int, error) {
	if str, ok := err.(string); ok {
		return ctx.ctx.Write(append([]byte{cmd_error}, []byte(str)...))
	} else if e, ok := err.(error); ok {
		return ctx.ctx.Write(append([]byte{cmd_error}, []byte(e.Error())...))
	}
	return ctx.ctx.Write(append([]byte{cmd_error}, []byte(fmt.Sprintf("%v", err))...))
}
