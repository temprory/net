package main

import (
	"github.com/temprory/event"
	"github.com/temprory/log"
	"github.com/temprory/net"
	"time"
)

const (
	CMD_BROAD = uint32(1)
	EVT_BROAD = "broadcast"
)

var (
	eventMgr = event.New("broadcast")
)

func onNewClient(client net.ITcpClient) {
	log.Info("onNewClient")
	//订阅广播
	eventMgr.Subscrib(client.Index(), EVT_BROAD, func(e interface{}, args ...interface{}) {
		if len(args) == 1 {
			if msg, ok := args[0].(net.IMessage); ok {
				if err := client.SendMsg(msg); err != nil {
					log.Info("server send to %v: %v failed: %v", client.Conn().RemoteAddr().String(), string(msg.Body()), err)
				}
			}
		}
	})
	//断开时取消订阅广播
	client.OnClose("-broadcast", func(c net.ITcpClient) {
		eventMgr.Unsubscrib(client.Index())
	})
}

//广播任务
func goBroadcast() {
	go func() {
		for {
			time.Sleep(time.Second)
			go eventMgr.Publish(EVT_BROAD, net.NewMessage(CMD_BROAD, []byte("broadcast")))
		}
	}()
}

func main() {
	server := net.NewTcpServer("echo")

	//应该在登录成功后再注册，示例简化，这里放在连接成功时
	server.HandleNewClient(onNewClient)

	goBroadcast()

	server.Serve(":8888", time.Second*5)
}
