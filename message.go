package net

import (
	"encoding/binary"
	"strconv"
	"strings"
)

type asyncMessage struct {
	data []byte
	cb   func(*TcpClient, error)
}

type wsAsyncMessage struct {
	data []byte
	cb   func(*WSClient, error)
}

const (
	DEFAULT_MESSAGE_HEAD_LEN int = 16

	DEFAULT_BODY_LEN_IDX_BEGIN int = 0
	DEFAULT_BODY_LEN_IDX_END   int = 4

	DEFAULT_CMD_IDX_BEGIN int = 4
	DEFAULT_CMD_IDX_END   int = 8

	DEFAULT_EXT_IDX_BEGIN int = 8
	DEFAULT_EXT_IDX_END   int = 16

	CmdFlagMaskGzip = uint32(1) << 31
	CmdPing         = uint32(0x1 << 24)
	CmdSetReaIp     = uint32(0x1<<24 + 1)
	CmdRpcMethod    = uint32(0x1<<24 + 2)
	CmdRpcError     = uint32(0x1<<24 + 3)
	CmdUserMax      = 0xFFFFFF
)

// type IMessage interface {
// 	HeadLen() int
// 	BodyLen() int

// 	Cmd() uint32
// 	SetCmd(cmd uint32)

// 	Ext() uint64
// 	SetExt(ext uint64)

// 	RpcSeq() int64
// 	SetRpcSeq(seq int64)

// 	Data() []byte
// 	SetData(data []byte)

// 	RawData() []byte
// 	SetRawData(rawData []byte)

// 	Body() []byte
// 	SetBody(body []byte)

// 	Encrypt(seq int64, key uint32, cipher ICipher) []byte
// 	Decrypt(seq int64, key uint32, cipher ICipher) ([]byte, error)
// }

type Message struct {
	data    []byte
	rawData []byte
}

func (msg *Message) HeadLen() int {
	return DEFAULT_MESSAGE_HEAD_LEN
}

func (msg *Message) BodyLen() int {
	return int(binary.LittleEndian.Uint32(msg.data[DEFAULT_BODY_LEN_IDX_BEGIN:DEFAULT_BODY_LEN_IDX_END]))
}

func (msg *Message) Cmd() uint32 {
	return binary.LittleEndian.Uint32(msg.data[DEFAULT_CMD_IDX_BEGIN:DEFAULT_CMD_IDX_END])
}

func (msg *Message) SetCmd(cmd uint32) {
	binary.LittleEndian.PutUint32(msg.data[DEFAULT_CMD_IDX_BEGIN:DEFAULT_CMD_IDX_END], cmd)
}

func (msg *Message) Ext() uint64 {
	return binary.LittleEndian.Uint64(msg.data[DEFAULT_EXT_IDX_BEGIN:DEFAULT_EXT_IDX_END])
}

func (msg *Message) SetExt(ext uint64) {
	binary.LittleEndian.PutUint64(msg.data[DEFAULT_EXT_IDX_BEGIN:DEFAULT_EXT_IDX_END], ext)
}

func (msg *Message) RpcSeq() int64 {
	return int64(binary.LittleEndian.Uint64(msg.data[DEFAULT_EXT_IDX_BEGIN:DEFAULT_EXT_IDX_END]))
}

func (msg *Message) SetRpcSeq(seq int64) {
	binary.LittleEndian.PutUint64(msg.data[DEFAULT_EXT_IDX_BEGIN:DEFAULT_EXT_IDX_END], uint64(seq))
}

func (msg *Message) Data() []byte {
	return msg.data
}

func (msg *Message) SetData(data []byte) {
	msg.data = data
}

func (msg *Message) RawData() []byte {
	return msg.rawData
}

func (msg *Message) SetRawData(rawData []byte) {
	msg.rawData = rawData
}

func (msg *Message) Body() []byte {
	return msg.data[DEFAULT_MESSAGE_HEAD_LEN:]
}

func (msg *Message) SetBody(data []byte) {
	needLen := DEFAULT_MESSAGE_HEAD_LEN + len(data) - len(msg.data)
	if needLen > 0 {
		msg.data = append(msg.data, make([]byte, needLen)...)
	} else if needLen < 0 {
		msg.data = msg.data[:DEFAULT_MESSAGE_HEAD_LEN+len(data)]
	}
	copy(msg.data[DEFAULT_MESSAGE_HEAD_LEN:], data)
	binary.LittleEndian.PutUint32(msg.data[DEFAULT_BODY_LEN_IDX_BEGIN:DEFAULT_BODY_LEN_IDX_END], uint32(len(data)))
}

func (msg *Message) Encrypt(seq int64, key uint32, cipher ICipher) []byte {
	if cipher != nil {
		msg.rawData = cipher.Encrypt(seq, key, msg.data)
	} else {
		msg.rawData = msg.data
	}
	return msg.rawData
}

func (msg *Message) Decrypt(seq int64, key uint32, cipher ICipher) ([]byte, error) {
	var err error
	if cipher != nil {
		msg.data, err = cipher.Decrypt(seq, key, msg.rawData)
	} else {
		msg.data = msg.rawData
	}
	return msg.data, err
}

func NewMessage(cmd uint32, data []byte) *Message {
	msg := &Message{
		data: make([]byte, len(data)+DEFAULT_MESSAGE_HEAD_LEN),
	}
	binary.LittleEndian.PutUint32(msg.data[DEFAULT_CMD_IDX_BEGIN:DEFAULT_CMD_IDX_END], cmd)
	binary.LittleEndian.PutUint32(msg.data[DEFAULT_BODY_LEN_IDX_BEGIN:DEFAULT_BODY_LEN_IDX_END], uint32(len(data)))
	if len(data) > 0 {
		copy(msg.data[DEFAULT_MESSAGE_HEAD_LEN:], data)
	}
	return msg
}

func RawMessage(data []byte) *Message {
	return &Message{
		data: data,
	}
}

func NewRpcMessage(cmd uint32, seq int64, data []byte) *Message {
	msg := &Message{
		data: make([]byte, len(data)+DEFAULT_MESSAGE_HEAD_LEN),
	}
	binary.LittleEndian.PutUint32(msg.data[DEFAULT_CMD_IDX_BEGIN:DEFAULT_CMD_IDX_END], cmd)
	binary.LittleEndian.PutUint64(msg.data[DEFAULT_EXT_IDX_BEGIN:DEFAULT_EXT_IDX_END], uint64(seq))
	binary.LittleEndian.PutUint32(msg.data[DEFAULT_BODY_LEN_IDX_BEGIN:DEFAULT_BODY_LEN_IDX_END], uint32(len(data)))
	if len(data) > 0 {
		copy(msg.data[DEFAULT_MESSAGE_HEAD_LEN:], data)
	}
	return msg
}

func RealIpMsg(ip string) *Message {
	ret := strings.Split(ip, ".")
	if len(ret) == 4 {
		var err error
		var ipv int
		var data = make([]byte, 4)
		for i, v := range ret {
			ipv, err = strconv.Atoi(v)
			if err != nil || ipv > 255 {
				return nil
			}
			data[i] = byte(ipv)
		}
		return NewMessage(CmdSetReaIp, data)
	}
	return NewMessage(CmdSetReaIp, []byte(ip))
}

func PingMsg() *Message {
	return NewMessage(CmdPing, nil)
}
