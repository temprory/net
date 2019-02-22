package net

import (
	"encoding/binary"
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
	if len(data) <= cipher.threshold {
		return data
	}
	body := gzipCompress(data[_message_head_len:])
	newData := append(make([]byte, _message_head_len), body...)
	copy(newData[:_message_head_len], data[:_message_head_len])
	cmd := binary.LittleEndian.Uint32(newData[_cmd_idx_begin:_cmd_idx_end])
	binary.LittleEndian.PutUint32(newData[_cmd_idx_begin:_cmd_idx_end], cmd|CmdFlagMaskGzip)
	binary.LittleEndian.PutUint32(newData[_bodylen_idx_begin:_bodylen_idx_end], uint32(len(body)))
	return newData
}

func (cipher *CipherGzip) Decrypt(seq int64, key uint32, data []byte) ([]byte, error) {
	cmd := binary.LittleEndian.Uint32(data[_cmd_idx_begin:_cmd_idx_end])
	if cmd&CmdFlagMaskGzip != CmdFlagMaskGzip {
		return data, nil
	}
	binary.LittleEndian.PutUint32(data[_cmd_idx_begin:_cmd_idx_end], cmd&(^CmdFlagMaskGzip))
	body, err := gzipUnCompress(data[_message_head_len:])
	if err == nil {
		return append(data[:_message_head_len], body...), nil
	}
	return nil, err
}

func NewCipherGzip(threshold int) ICipher {
	if threshold <= 0 {
		threshold = 1024
	}
	return &CipherGzip{threshold}
}
