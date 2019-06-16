package net

import (
	"context"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/temprory/graceful"
	"github.com/temprory/log"
	"github.com/temprory/util"
	"net/http"
	"os"
	"sync/atomic"
	"time"
)

// func WsPage(c *gin.Context) {
// 	// change the reqest to websocket model
// 	conn, error := (&websocket.Upgrader{CheckOrigin: func(r *http.Request) bool { return true }}).Upgrade(c.Writer, c.Request, nil)
// 	if error != nil {
// 		http.NotFound(c.Writer, c.Request)
// 		return
// 	}
// 	// websocket connect
// 	client := &ws.WSClient{Id: uuid.NewV4().String(), Socket: conn, Send: make(chan []byte)}

// 	ws.Manager.Register <- client

// 	go client.Read()
// 	go client.Write()
// }

type WSServer struct {
	*WSEngine

	// router
	*gin.Engine

	// http server
	*graceful.HttpServer

	// websocket upgrader
	upgrader *websocket.Upgrader

	// http请求过滤
	requestHandler func(ctx *gin.Context) error

	// ws连接成功
	connectHandler func(cli *WSClient, ctx *gin.Context) error

	// 连接断开
	disconnectHandler func(cli *WSClient, ctx *gin.Context)

	// 当前连接数
	OnlineNum int64

	// 过载保护,同时最大连接数
	MaxOnline int64

	// handlers map[uint32]func(cli *WSClient, cmd uint32, data []byte)

	clients map[*WSClient]struct{}
}

func (s *WSServer) SetUpgrader(upgrader *websocket.Upgrader) {
	s.upgrader = upgrader
}

// func (s *WSServer) deleClient(cli *WSClient) {
// 	s.Lock()
// 	delete(s.clients, cli)
// 	s.Unlock()
// 	// 计数减
// 	atomic.AddInt64(&s.OnlineNum, -1)
// }

func (s *WSServer) upgrade(ctx *gin.Context) {
	defer handlePanic()

	if s.shutdown {
		http.NotFound(ctx.Writer, ctx.Request)
		return
	}

	// http升级ws请求的应用层过滤
	if s.requestHandler != nil && s.requestHandler(ctx) != nil {
		http.NotFound(ctx.Writer, ctx.Request)
		return
	}

	conn, err := s.upgrader.Upgrade(ctx.Writer, ctx.Request, nil)
	if err != nil {
		http.NotFound(ctx.Writer, ctx.Request)
		return
	}

	// 计数减
	online := atomic.AddInt64(&s.OnlineNum, 1)

	//过载保护,大于配置的最大连接数则拒绝连接
	if s.MaxOnline > 0 && online >= s.MaxOnline {
		atomic.AddInt64(&s.OnlineNum, -1)
		http.NotFound(ctx.Writer, ctx.Request)
		return
	}

	// 	// websocket connect
	// 	client := &ws.WSClient{Id: uuid.NewV4().String(), Socket: conn, Send: make(chan []byte)}
	var cli = newClient(conn, s.WSEngine)
	s.Lock()
	s.clients[cli] = struct{}{}
	s.Unlock()

	conn.SetReadLimit(s.ReadLimit)

	// 清理
	defer func() {
		s.Lock()
		delete(s.clients, cli)
		s.Unlock()
		// 计数减
		atomic.AddInt64(&s.OnlineNum, -1)

		cli.Stop()

		// ws关闭处理接口
		if s.disconnectHandler != nil {
			s.disconnectHandler(cli, ctx)
		}
	}()

	// ws新连接过滤接口
	if s.connectHandler != nil && s.connectHandler(cli, ctx) != nil {
		return
	}

	// 收到ping、pong更新读超时
	conn.SetPingHandler(func(string) error {
		if s.ReadTimeout > 0 {
			conn.SetReadDeadline(time.Now().Add(s.ReadTimeout))
		}
		return nil
	})
	conn.SetPongHandler(func(string) error {
		if s.ReadTimeout > 0 {
			conn.SetReadDeadline(time.Now().Add(s.ReadTimeout))
		}
		return nil
	})

	go cli.writeloop()

	cli.readloop()
}

// websocket upgrade handler
func (s *WSServer) onWebsocketRequest(ctx *gin.Context) {
	s.upgrade(ctx)
}

// 默认消息处理
// func (s *WSServer) onMessage(cli *WSClient, data []byte) {
// 	s.onMessage(cli, data)
// }

// 设置http升级ws请求的应用层过滤接口
func (s *WSServer) HandleRequest(h func(ctx *gin.Context) error) {
	s.requestHandler = h
}

// 设置ws新连接过滤接口
func (s *WSServer) HandleConnect(h func(cli *WSClient, ctx *gin.Context) error) {
	s.connectHandler = h
}

// 设置ws关闭处理接口
func (s *WSServer) HandleDisconnect(h func(cli *WSClient, ctx *gin.Context)) {
	s.disconnectHandler = h
}

// ws路由
func (s *WSServer) HandleWs(path string) {
	s.Any(path, s.onWebsocketRequest)
}

func (s *WSServer) Shutdown(timeout time.Duration, cb func(error)) {
	s.Lock()
	shutdown := s.shutdown
	s.shutdown = true
	s.Unlock()
	if !shutdown {
		log.Debug("WSEngine Shutdown ...")

		if timeout <= 0 {
			timeout = DefaultShutdownTimeout
		}

		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		defer cancel()

		done := make(chan struct{}, 1)
		util.Go(func() {
			s.Wait()

			s.Lock()
			for cli := range s.clients {
				cli.Stop()
			}
			s.Unlock()

			s.HttpServer.Shutdown()

			done <- struct{}{}
		})

		select {
		case <-ctx.Done():
			log.Debug("WSEngine Shutdown timeout")
			if cb != nil {
				cb(ErrWSEngineShutdownTimeout)
			}
		case <-done:
			log.Debug("WSEngine Shutdown success")
			cb(nil)
		}
	}
}

// ws server factory
func NewWebsocketServer(name string, addr string) (*WSServer, error) {
	var err error
	svr := &WSServer{
		Engine:    gin.New(),
		WSEngine:  NewWebsocketEngine(),
		MaxOnline: DefaultMaxOnline,
		upgrader:  &websocket.Upgrader{CheckOrigin: func(r *http.Request) bool { return true }},
		clients:   map[*WSClient]struct{}{},
	}

	// svr.WSEngine.MessageHandler = svr.onMessage

	svr.HttpServer, err = graceful.NewHttpServer(addr, svr.Engine, time.Second*5, nil, func() {
		os.Exit(-1)
	})

	//svr.HttpServer.Shutdown()

	return svr, err
}
