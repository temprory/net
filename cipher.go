package net

import (
	"encoding/binary"
)

const (
	CipherGzipAll  = 0
	CipherGzipNone = -1

	DefaultThreshold = CipherGzipNone
)

type ICipher interface {
	Init()
	Encrypt(seq int64, key uint32, data []byte) []byte
	Decrypt(seq int64, key uint32, data []byte) ([]byte, error)
}

type CipherGzip struct {
	threshold int
}

func (cipher *CipherGzip) Init() {

}

func (cipher *CipherGzip) Encrypt(seq int64, key uint32, data []byte) []byte {
	if cipher.threshold < 0 || (len(data) <= cipher.threshold+DEFAULT_MESSAGE_HEAD_LEN) {
		return data
	}
	body := gzipCompress(data[DEFAULT_MESSAGE_HEAD_LEN:])
	newData := append(make([]byte, DEFAULT_MESSAGE_HEAD_LEN), body...)
	copy(newData[:DEFAULT_MESSAGE_HEAD_LEN], data[:DEFAULT_MESSAGE_HEAD_LEN])
	cmd := binary.LittleEndian.Uint32(newData[DEFAULT_CMD_IDX_BEGIN:DEFAULT_CMD_IDX_END])
	binary.LittleEndian.PutUint32(newData[DEFAULT_CMD_IDX_BEGIN:DEFAULT_CMD_IDX_END], cmd|CmdFlagMaskGzip)
	binary.LittleEndian.PutUint32(newData[DEFAULT_BODY_LEN_IDX_BEGIN:DEFAULT_BODY_LEN_IDX_END], uint32(len(body)))
	return newData
}

func (cipher *CipherGzip) Decrypt(seq int64, key uint32, data []byte) ([]byte, error) {
	cmd := binary.LittleEndian.Uint32(data[DEFAULT_CMD_IDX_BEGIN:DEFAULT_CMD_IDX_END])

	if cmd&CmdFlagMaskGzip != CmdFlagMaskGzip {
		return data, nil
	}
	binary.LittleEndian.PutUint32(data[DEFAULT_CMD_IDX_BEGIN:DEFAULT_CMD_IDX_END], cmd&(^CmdFlagMaskGzip))
	body, err := gzipUnCompress(data[DEFAULT_MESSAGE_HEAD_LEN:])
	if err == nil {
		return append(data[:DEFAULT_MESSAGE_HEAD_LEN], body...), nil
	}

	return nil, err
}

func NewCipherGzip(threshold int) ICipher {
	// if threshold <= 0 {
	// 	threshold = 1024
	// }
	return &CipherGzip{threshold}
}
