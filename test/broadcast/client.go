package main

import (
	"github.com/temprory/log"
	"github.com/temprory/net"
)

const (
	CMD_BROAD = uint32(1)
)

func onBroadcast(client *net.TcpClient, msg net.IMessage) {
	str := "aaaaaaaaaa"
	for j := 0; j < 256; j++ {
		str += "aaaaaaaaaa"
	}
	if str == string(msg.Body()) {
		log.Debug("client onBroadcast recv from %v: %v", client.Conn.RemoteAddr().String(), "aaaaaa")
	} else {
		log.Debug("client onBroadcast recv from %v: %v", client.Conn.RemoteAddr().String(), string(msg.Body()))
	}

}

func runClient() {
	var (
		err        error
		addr       = "127.0.0.1:8888"
		client     *net.TcpClient
		cipher     net.ICipher = nil
		autoReconn             = true
		netengine              = net.NewTcpEngine()
	)

	netengine.Handle(CMD_BROAD, onBroadcast)
	cipher = net.NewCipherGzip(net.DefaultThreshold)
	netengine.HandleNewCipher(func() net.ICipher {
		return cipher
	})
	client, err = net.NewTcpClient(addr, netengine, cipher, autoReconn, nil)
	if err != nil {
		log.Debug("NewTcpClient failed: %v, %v", client, err)
	}
}

func main() {
	runClient()
	<-make(chan int)
}
