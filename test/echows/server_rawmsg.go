package main

import (
	"github.com/temprory/log"
	"github.com/temprory/net"
	"runtime"
	"time"
)

const (
	CMD_ECHO = uint32(1)
)

func onMessage(client *net.WSClient, msg *net.Message) {
	//log.Info("server recv from %v: %v", client.Conn.RemoteAddr().String(), string(msg.Data()))
	//err := client.SendMsg(msg)
	//log.Info("server send   to %s: %v, %v", client.Conn.RemoteAddr().String(), string(msg.Data()), err)
	client.SendMsg(msg)
}

func main() {
	server, err := net.NewWebsocketServer("echo", ":8888")
	if err != nil {
		log.Panic("websocket.NewServer failed: %v", err)
	}
	server.HandleWs("/ws/echo")
	server.HandleMessage(onMessage)

	go func() {
		for {
			time.Sleep(time.Second)
			log.Println("--- runtime.NumGoroutine():", runtime.NumGoroutine())
			log.Println("    client num:", server.ClientNum())
		}

	}()
	server.Serve()

}
