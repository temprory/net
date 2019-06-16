package main

import (
	"github.com/temprory/log"
	"github.com/temprory/net"
	"time"
)

const (
	CMD_ECHO = uint32(1)
)

func onEcho(client *net.WSClient, msg net.IMessage) {
	//cli.SendMsg(cmd, data)
	log.Info("onEcho, cmd: %v, data: %v", msg.Cmd(), string(msg.Body()))
}

func main() {
	cli, err := net.NewWebsocketClient("ws://localhost:8888/ws/echo")
	if err != nil {
		log.Panic("websocket.NewServer failed: %v, %v", err, time.Now())
	}

	cli.Handle(CMD_ECHO, onEcho)
	s := ""
	for i := 0; i < 2048; i++ {
		s += "a"
	}
	for i := 0; i < 10000000; i++ {
		cli.SendMsg(net.NewMessage(CMD_ECHO, []byte(s)))
		time.Sleep(time.Second)
		// cli.Stop()
		// time.Sleep(time.Second)
		break
	}
}
