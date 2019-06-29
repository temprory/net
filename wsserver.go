package net

import (
	//"context"
	//"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/temprory/graceful"
	"github.com/temprory/log"
	"github.com/temprory/util"
	"net/http"
	"os"
	"sync/atomic"
	"time"
)

type WSServer struct {
	*WSEngine

	// router
	//*gin.Engine

	// http server
	*graceful.HttpServer

	// websocket upgrader
	upgrader *websocket.Upgrader

	// http请求过滤
	requestHandler func(w http.ResponseWriter, r *http.Request) error

	// ws连接成功
	connectHandler func(cli *WSClient, w http.ResponseWriter, r *http.Request) error

	// 连接断开
	disconnectHandler func(cli *WSClient, w http.ResponseWriter, r *http.Request)

	// 当前连接数
	currLoad int64

	// 过载保护,同时最大连接数
	maxLoad int64

	// handlers map[uint32]func(cli *WSClient, cmd uint32, data []byte)

	clients map[*WSClient]struct{}

	routes map[string]func(http.ResponseWriter, *http.Request)
}

func (s *WSServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if h, ok := s.routes[r.URL.Path]; ok {
		h(w, r)
	} else {
		http.NotFound(w, r)
	}
}

func (s *WSServer) SetUpgrader(upgrader *websocket.Upgrader) {
	s.upgrader = upgrader
}

// func (s *WSServer) deleClient(cli *WSClient) {
// 	s.Lock()
// 	delete(s.clients, cli)
// 	s.Unlock()
// 	// 计数减
// 	atomic.AddInt64(&s.currLoad, -1)
// }

func (s *WSServer) onWebsocketRequest(w http.ResponseWriter, r *http.Request) {
	defer handlePanic()

	if s.shutdown {
		http.NotFound(w, r)
		return
	}

	// http升级ws请求的应用层过滤
	if s.requestHandler != nil && s.requestHandler(w, r) != nil {
		http.NotFound(w, r)
		return
	}

	// 计数减
	online := atomic.AddInt64(&s.currLoad, 1)

	//过载保护,大于配置的最大连接数则拒绝连接
	if s.maxLoad > 0 && online > s.maxLoad {
		atomic.AddInt64(&s.currLoad, -1)
		http.NotFound(w, r)
		return
	}

	conn, err := s.upgrader.Upgrade(w, r, nil)
	if err != nil {
		http.NotFound(w, r)
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
		atomic.AddInt64(&s.currLoad, -1)

		cli.Stop()

		// ws关闭处理接口
		if s.disconnectHandler != nil {
			s.disconnectHandler(cli, w, r)
		}
	}()

	// ws新连接过滤接口
	if s.connectHandler != nil && s.connectHandler(cli, w, r) != nil {
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
// func (s *WSServer) onWebsocketRequest(w http.ResponseWriter, r *http.Request) {
// 	s.upgrade(ctx)
// }

// 默认消息处理
// func (s *WSServer) onMessage(cli *WSClient, data []byte) {
// 	s.onMessage(cli, data)
// }

func (server *WSServer) CurrLoad() int64 {
	return atomic.LoadInt64(&server.currLoad)
}

func (server *WSServer) MaxLoad() int64 {
	return server.maxLoad
}

func (server *WSServer) SetMaxConcurrent(maxLoad int64) {
	server.maxLoad = maxLoad
}

func (s *WSServer) ClientNum() int {
	s.Lock()
	defer s.Unlock()
	return len(s.clients)
}

func (s *WSServer) HandleRequest(h func(w http.ResponseWriter, r *http.Request) error) {
	s.requestHandler = h
}

// 设置ws新连接过滤接口
func (s *WSServer) HandleConnect(h func(cli *WSClient, w http.ResponseWriter, r *http.Request) error) {
	s.connectHandler = h
}

// 设置ws关闭处理接口
func (s *WSServer) HandleDisconnect(h func(cli *WSClient, w http.ResponseWriter, r *http.Request)) {
	s.disconnectHandler = h
}

// ws路由
func (s *WSServer) HandleWs(path string) {
	s.routes[path] = s.onWebsocketRequest
}

// http路由
func (s *WSServer) HandleHttp(path string, h func(w http.ResponseWriter, r *http.Request)) {
	s.routes[path] = h
}

func (s *WSServer) stopClients() {
	s.Lock()
	defer s.Unlock()

	for client, _ := range s.clients {
		//client.CancelOnClose(_client_rm_from_server)
		client.Stop()
	}
}

func (s *WSServer) Shutdown(timeout time.Duration, cb func(error)) {
	s.Lock()
	shutdown := s.shutdown
	s.shutdown = true
	s.Unlock()
	if !shutdown {
		log.Debug("WSServer Shutdown ...")

		if timeout <= 0 {
			timeout = DefaultShutdownTimeout
		}

		done := make(chan struct{}, 1)
		util.Go(func() {
			s.Wait()

			s.stopClients()

			s.HttpServer.Shutdown()

			done <- struct{}{}
		})

		select {
		case <-time.After(timeout):
			log.Debug("WSServer Shutdown timeout")
			if cb != nil {
				cb(ErrWSEngineShutdownTimeout)
			}
		case <-done:
			log.Debug("WSServer Shutdown success")
			cb(nil)
		}
	}
}

// ws server factory
func NewWebsocketServer(name string, addr string) (*WSServer, error) {
	var err error
	svr := &WSServer{
		//Engine:    gin.New(),
		WSEngine: NewWebsocketEngine(),
		maxLoad:  DefaultMaxOnline,
		upgrader: &websocket.Upgrader{CheckOrigin: func(r *http.Request) bool { return true }},
		clients:  map[*WSClient]struct{}{},
		routes:   map[string]func(http.ResponseWriter, *http.Request){},
	}

	// svr.WSEngine.MessageHandler = svr.onMessage

	svr.HttpServer, err = graceful.NewHttpServer(addr, svr, time.Second*5, nil, func() {
		os.Exit(-1)
	})

	//svr.HttpServer.Shutdown()

	return svr, err
}
