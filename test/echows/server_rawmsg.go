package main

import (
	"github.com/temprory/log"
	"github.com/temprory/net"
	"github.com/temprory/util"
	"os"
	"runtime"
	"syscall"
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
		log.Panic("NewWebsocketServer failed: %v", err)
	}
	server.SetMaxConcurrent(500)
	server.HandleWs("/ws/echo")
	server.HandleMessage(onMessage)

	go func() {
		for {
			time.Sleep(time.Second)
			log.Println("--- runtime.NumGoroutine():", runtime.NumGoroutine())
			log.Println("    client num:", server.ClientNum())
		}

	}()

	go server.Serve()

	util.HandleSignal(func(sig os.Signal) {
		if sig == syscall.SIGTERM || sig == syscall.SIGINT {
			server.Shutdown(time.Second*5, func(err error) {
				log.Info("--- shutdown: %v", err)
				os.Exit(-1)
			})
		}
	})
}
