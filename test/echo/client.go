package main

import (
	"fmt"
	"github.com/temprory/log"
	"github.com/temprory/net"
	"sync"
	"time"
)

var (
	CMD_ECHO = uint32(1)
)

func onEcho(client *net.TcpClient, msg *net.Message) {
	log.Debug("client onEcho recv from %v: %v", client.Conn.RemoteAddr().String(), string(msg.Body()))
}

func main() {
	var (
		addr                   = "127.0.0.1:8888"
		cipher     net.ICipher = net.NewCipherGzip(-1)
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
