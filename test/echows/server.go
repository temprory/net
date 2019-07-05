package main

import (
	"github.com/temprory/log"
	"github.com/temprory/net"
)

const (
	CMD_ECHO = uint32(1)
)

func onEcho(client *net.WSClient, msg net.IMessage) {
	log.Info("server recv from %v: %v", client.Conn.RemoteAddr().String(), string(msg.Body()))
	err := client.SendMsg(msg)
	log.Info("server send   to %s: %v, %v", client.Conn.RemoteAddr().String(), string(msg.Body()), err)

}

func main() {
	server, err := net.NewWebsocketServer("echo", ":8888")
	if err != nil {
		log.Panic("NewWebsocketServer failed: %v", err)
	}
	server.HandleWs("/ws/echo")
	server.Handle(CMD_ECHO, onEcho)

	server.Serve()
}
