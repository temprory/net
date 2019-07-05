package main

import (
	"encoding/binary"
	"github.com/temprory/crypto"
	"github.com/temprory/log"
	"github.com/temprory/net"
	"github.com/temprory/util"
	"runtime"
	"time"
)

var ()

const (
	CMD_ECHO = uint32(1)
)

func onEcho(client *net.TcpClient, msg net.IMessage) {
	// log.Info("server onEcho recv from %v: %v", client.Conn.RemoteAddr().String(), string(msg.Body()))
	// err := client.SendMsg(msg)
	// log.Info("server send to%s: %v, %v,", client.Conn.RemoteAddr().String(), string(msg.Body()), err)
	client.SendMsg(msg)
}

var (
	aeskey = []byte("12345678123456781234567812345678")
	aesiv  = []byte("12345678123456781234567812345678")
)

type CipherGzipAes struct {
	threshold int
}

func (cipher *CipherGzipAes) Init() {

}

func (cipher *CipherGzipAes) Encrypt(seq int64, key uint32, data []byte) []byte {
	if cipher.threshold < 0 || (len(data) <= cipher.threshold+net.DEFAULT_MESSAGE_HEAD_LEN) {
		return data
	}
	log.Debug("--- Encrypt")
	body := util.GZipCompress(data[16:])
	body, _ = crypto.AesCBCEncrypt(aeskey, aesiv, body)
	newData := append(make([]byte, net.DEFAULT_MESSAGE_HEAD_LEN), body...)
	copy(newData[:net.DEFAULT_MESSAGE_HEAD_LEN], data[:net.DEFAULT_MESSAGE_HEAD_LEN])
	cmd := binary.LittleEndian.Uint32(newData[net.DEFAULT_CMD_IDX_BEGIN:net.DEFAULT_CMD_IDX_END])
	binary.LittleEndian.PutUint32(newData[net.DEFAULT_CMD_IDX_BEGIN:net.DEFAULT_CMD_IDX_END], cmd|net.CmdFlagMaskGzip)
	binary.LittleEndian.PutUint32(newData[net.DEFAULT_BODY_LEN_IDX_BEGIN:net.DEFAULT_BODY_LEN_IDX_END], uint32(len(body)))
	return newData
}

func (cipher *CipherGzipAes) Decrypt(seq int64, key uint32, data []byte) ([]byte, error) {
	cmd := binary.LittleEndian.Uint32(data[net.DEFAULT_CMD_IDX_BEGIN:net.DEFAULT_CMD_IDX_END])

	if cmd&net.CmdFlagMaskGzip != net.CmdFlagMaskGzip {
		return data, nil
	}
	log.Debug("--- Decrypt")
	binary.LittleEndian.PutUint32(data[net.DEFAULT_CMD_IDX_BEGIN:net.DEFAULT_CMD_IDX_END], cmd&(^net.CmdFlagMaskGzip))
	body, err := crypto.AesCBCDecrypt(aeskey, aesiv, data[16:])
	if err != nil {
		return nil, err
	}
	body, err = util.GZipUnCompress(body)
	if err != nil {
		return nil, err
	}

	return append(data[:net.DEFAULT_MESSAGE_HEAD_LEN], body...), nil
}

func NewCipherGzip(threshold int) net.ICipher {
	// if threshold <= 0 {
	// 	threshold = 1024
	// }
	return &CipherGzipAes{threshold}
}

func main() {
	cipher := NewCipherGzip(net.CipherGzipAll)
	server := net.NewTcpServer("echo")
	server.SetMaxConcurrent(500)
	server.Handle(CMD_ECHO, onEcho)
	server.HandleNewCipher(func() net.ICipher {
		return cipher
	})

	go func() {
		for {
			time.Sleep(time.Second)
			log.Println("--- runtime.NumGoroutine():", runtime.NumGoroutine())
			log.Println("    client num:", server.CurrLoad())
		}

	}()

	server.Serve(":8888", time.Second*5)
}
