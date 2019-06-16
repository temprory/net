package main

import (
	"github.com/temprory/log"
	"github.com/temprory/net"
	"time"
)

var (
	CMD_ECHO = uint32(1)

	reqData = make([]byte, 2048)
)

func onEcho(client net.ITcpClient, msg net.IMessage) {
	log.Debug("client onEcho recv from %v: %v, %v", client.Conn().RemoteAddr().String(), string(reqData) == string(msg.Body()), len(msg.Body()))
}

func onConnected(client net.ITcpClient) {
	for i := 0; i < 20; i++ {
		err := client.SendMsg(net.NewMessage(CMD_ECHO, reqData))
		if err != nil {
			break
		}
		time.Sleep(time.Second)
		break
	}
}

func main() {
	var (
		err        error
		addr       = "127.0.0.1:18200"
		client     net.ITcpClient
		cipher     net.ICipher = net.NewCipherGzip(-1)
		autoReconn             = true
		netengine              = net.NewTcpEngine()
	)

	for i, _ := range reqData {
		reqData[i] = 'a'
	}

	netengine.Handle(CMD_ECHO, onEcho)

	client, err = net.NewTcpClient(addr, netengine, cipher, autoReconn, onConnected)
	if err != nil {
		log.Debug("NewTcpClient failed: %v, %v", client, err)
	}

	<-make(chan int)
}
