package rpc

import (
	"errors"
	"fmt"
	"github.com/valyala/fastrpc"
	"github.com/valyala/fastrpc/tlv"
	"net"
	"time"
)

func newTestResponse() fastrpc.ResponseReader {
	return &tlv.Response{}
}

type Client struct {
	c     *fastrpc.Client
	codec IRpcCodec
}

func (c *Client) Call(method string, req interface{}, rsp interface{}, timeout time.Duration) error {
	if timeout <= 0 {
		return fmt.Errorf("invalid timeout arg: %v", timeout)
	}

	var (
		data []byte
		err  error
	)

	if req != nil {
		data, err = c.codec.Marshal(req)
		if err != nil {
			return err
		}
	}

	var tlvReq tlv.Request
	var tlvRsp tlv.Response

	pack := make([]byte, 1+len(method)+len(data))
	pack[0] = byte(len(method))

	copy(pack[1:], []byte(method))
	copy(pack[1+len(method):], data)

	tlvReq.SwapValue(pack)
	err = c.c.DoDeadline(&tlvReq, &tlvRsp, time.Now().Add(timeout))
	if err != nil {
		return err
	}
	data = tlvRsp.Value()
	if len(data) < 1 {
		return errors.New("unexpected error")
	}

	switch data[0] {
	case cmd_msg:
		if len(data) > 1 && rsp != nil {
			if prsp, ok := rsp.(*string); ok {
				*prsp = string(data[1:])
			} else {
				c.codec.Unmarshal(data[1:], rsp)
			}
		}
		return nil
	case cmd_error:
		if len(data) > 1 {
			return errors.New(string(data[1:]))
		}
		return errors.New("unexpected error")
	}
	return errors.New("unexpected error")
}

func NewClient(addr string) *Client {
	c := &Client{
		c: &fastrpc.Client{
			NewResponse: func() fastrpc.ResponseReader {
				return &tlv.Response{}
			},
			Dial: func(string) (net.Conn, error) {
				return net.Dial("tcp", addr)
			},
		},
		codec: DefaultRpcCodec,
	}
	return c
}
