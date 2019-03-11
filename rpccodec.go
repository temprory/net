package net

import (
	"bytes"
	"encoding/gob"
	"github.com/golang/protobuf/proto"
	"github.com/vmihailenco/msgpack"
)

var (
	defaultRpcCodecType = "json"
	DefaultRpcCodec     = &RpcCodecJson{}
)

type IRpcCodec interface {
	Marshal(v interface{}) ([]byte, error)
	Unmarshal(data []byte, v interface{}) error
}

type RpcCodecGob struct{}

func (c *RpcCodecGob) Marshal(v interface{}) ([]byte, error) {
	buffer := &bytes.Buffer{}
	err := gob.NewEncoder(buffer).Encode(v)
	if err != nil {
		return nil, err
	}
	return buffer.Bytes(), nil
}

func (c *RpcCodecGob) Unmarshal(data []byte, v interface{}) error {
	return gob.NewDecoder(bytes.NewBuffer(data)).Decode(v)
}

type RpcCodecJson struct{}

func (c *RpcCodecJson) Marshal(v interface{}) ([]byte, error) {
	return json.Marshal(v)
}

func (c *RpcCodecJson) Unmarshal(data []byte, v interface{}) error {
	return json.Unmarshal(data, v)
}

type RpcCodecMsgpack struct{}

func (c *RpcCodecMsgpack) Marshal(v interface{}) ([]byte, error) {
	return msgpack.Marshal(v)
}

func (c *RpcCodecMsgpack) Unmarshal(data []byte, v interface{}) error {
	return msgpack.Unmarshal(data, v)
}

type RpcCodecProtobuf struct{}

func (c *RpcCodecProtobuf) Marshal(v interface{}) ([]byte, error) {
	msg, ok := v.(proto.Message)
	if ok {
		return proto.Marshal(msg)
	}
	return nil, ErrorRpcInvalidPbMessage
}

func (c *RpcCodecProtobuf) Unmarshal(data []byte, v interface{}) error {
	msg, ok := v.(proto.Message)
	if ok {
		return proto.Unmarshal(data, msg)
	}
	return ErrorRpcInvalidPbMessage
}
