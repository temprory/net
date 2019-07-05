package main

import (
	"encoding/binary"
	"fmt"
	"github.com/temprory/crypto"
	"github.com/temprory/log"
	"github.com/temprory/net"
	"github.com/temprory/util"
	"sync"
	"time"
)

var (
	CMD_ECHO = uint32(1)
)

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

func onEcho(client *net.TcpClient, msg net.IMessage) {
	log.Debug("client onEcho recv from %v: %v", client.Conn.RemoteAddr().String(), string(msg.Body()))
}

func main() {
	var (
		addr                   = "127.0.0.1:8888"
		cipher     net.ICipher = NewCipherGzip(-1)
		autoReconn             = false
		netengine              = net.NewTcpEngine()
	)

	netengine.Handle(CMD_ECHO, onEcho)

	wg := sync.WaitGroup{}

	for i := 0; i < 1000; i++ {
		wg.Add(1)

		go func() {
			defer wg.Done()

			client, err := net.NewTcpClient(addr, netengine, cipher, autoReconn, nil)
			if err != nil {
				log.Debug("NewTcpClient failed: %v, %v", client, err)
				return
			}

			for i := 0; true; i++ {
				err = client.SendMsg(net.NewMessage(CMD_ECHO, []byte(fmt.Sprintf("hello %v", i))))
				if err != nil {
					break
				}
				time.Sleep(time.Second)
				// break
			}

		}()
	}

	wg.Wait()
}
