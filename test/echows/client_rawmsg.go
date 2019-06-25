package main

import (
	"github.com/temprory/log"
	"github.com/temprory/net"
	"time"
)

const (
	CMD_ECHO = uint32(1)
)

func onMessage(client *net.WSClient, msg *net.Message) {
	//cli.SendMsg(cmd, data)
	log.Info("onMessage, data: %v", string(msg.Data()))
}

func main() {
	cli, err := net.NewWebsocketClient("ws://localhost:8888/ws/echo")
	if err != nil {
		log.Panic("websocket.NewServer failed: %v, %v", err, time.Now())
	}

	cli.HandleMessage(onMessage)
	s := "hello ws"
	// for i := 0; i < 2048; i++ {
	// 	s += "a"
	// }
	for {
		cli.SendMsg(net.RawMessage([]byte(s)))
		time.Sleep(time.Second)
		// cli.Stop()
		// time.Sleep(time.Second)
	}
}
