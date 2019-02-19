package main

import (
	"github.com/temprory/log"
	"github.com/temprory/net"
)

const (
	CMD_BROAD = uint32(1)
)

func onBroadcast(client net.ITcpClient, msg net.IMessage) {
	log.Debug("client onBroadcast recv from %v: %v", client.Conn().RemoteAddr().String(), string(msg.Body()))
}

func runClient() {
	var (
		err        error
		addr       = "127.0.0.1:8888"
		client     net.ITcpClient
		cipher     net.ICipher = nil
		autoReconn             = true
		netengine              = net.NewTcpEngine()
	)

	netengine.Handle(CMD_BROAD, onBroadcast)

	client, err = net.NewTcpClient(addr, netengine, cipher, autoReconn, nil)
	if err != nil {
		log.Debug("NewTcpClient failed: %v, %v", client, err)
	}
}

func main() {
	runClient()
	<-make(chan int)
}