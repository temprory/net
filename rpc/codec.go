package rpc

import (
	"gopkg.in/vmihailenco/msgpack.v2"
)

var (
	defaultRpcCodecType           = "msgpack"
	DefaultRpcCodec     IRpcCodec = &CodecMsgpack{}
)

type IRpcCodec interface {
	Marshal(v interface{}) ([]byte, error)
	Unmarshal(data []byte, v interface{}) error
}

type CodecMsgpack struct{}

func (c *CodecMsgpack) Marshal(v interface{}) ([]byte, error) {
	return msgpack.Marshal(v)
}

func (c *CodecMsgpack) Unmarshal(data []byte, v interface{}) error {
	return msgpack.Unmarshal(data, v)
}
