package main

import (
	"github.com/temprory/log"
	"github.com/temprory/net"
	"runtime"
	"time"
)

var ()

const (
	CMD_ECHO = uint32(1)
)

func onEcho(client *net.TcpClient, msg *net.Message) {
	// log.Info("server onEcho recv from %v: %v", client.Conn.RemoteAddr().String(), string(msg.Body()))
	// err := client.SendMsg(msg)
	// log.Info("server send to%s: %v, %v,", client.Conn.RemoteAddr().String(), string(msg.Body()), err)
	client.SendMsg(msg)
}

func main() {
	cipher := net.NewCipherGzip(-1)
	server := net.NewTcpServer("echo")
	server.SetMaxConcurrent(500)
	server.Handle(CMD_ECHO, onEcho)
	server.HandleNewCipher(func() net.ICipher {
		return cipher
	})

	go func() {
		for {
			time.Sleep(time.Second)
			log.Println("--- runtime.NumGoroutine():", runtime.NumGoroutine())
			log.Println("    client num:", server.CurrLoad())
		}

	}()

	server.Serve(":8888", time.Second*5)
}
