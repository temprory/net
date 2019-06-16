package net

import (
	"fmt"
	"net"
	"os"
	// "runtime"
	// "sync"
	"sync/atomic"
	"syscall"
	"time"
)

var (
	_client_rm_from_server = 0
)

type ITcpServer interface {
	ITcpEngin
	Start(addr string) error
	Stop()
	StopWithTimeout(timeout time.Duration, onStop func())
	Serve(addr string, stopTimeout time.Duration)
	CurrLoad() int64
	MaxLoad() int64
	SetMaxConcurrent(maxLoad int64)
	AcceptedNum() int64
	HandleServerStop(stopHandler func(server ITcpServer))
	EnableBroadcast()
	Broadcast(msg IMessage)
	BroadcastWithFilter(msg IMessage, filter func(*TcpClient) bool)
}

type TcpServer struct {
	TcpEngin
	tag string
	//running       bool
	enableBroad   bool
	addr          string
	accepted      int64
	currLoad      int64
	maxLoad       int64
	listener      *net.TCPListener
	stopTimeout   time.Duration
	onStop        func()
	onStopHandler func(server ITcpServer)
}

func (server *TcpServer) addClient(client *TcpClient) {
	if server.enableBroad {
		server.Lock()
		server.clients[client] = struct{}{}
		server.Unlock()

	}
	atomic.AddInt64(&server.currLoad, 1)
	server.OnNewClient(client)
}

func (server *TcpServer) deleClient(client *TcpClient) {
	if server.enableBroad {
		server.Lock()
		delete(server.clients, client)
		server.Unlock()
	}
	atomic.AddInt64(&server.currLoad, -1)
}

func (server *TcpServer) stopClients() {
	server.Lock()
	defer server.Unlock()

	for client, _ := range server.clients {
		client.CancelOnClose(_client_rm_from_server)
		client.Stop()
	}
}

func (server *TcpServer) listenerLoop() error {
	logDebug("[TcpServer %s] Running on: \"%s\"", server.tag, server.addr)
	defer logDebug("[TcpServer %s] Stopped.", server.tag)

	var (
		err error
		// idx    uint64
		conn   *net.TCPConn
		client *TcpClient
		// file      *os.File
		tempDelay time.Duration
	)
	for server.running {
		if conn, err = server.listener.AcceptTCP(); err == nil {
			if server.maxLoad == 0 || atomic.LoadInt64(&server.currLoad) < server.maxLoad {
				// if runtime.GOOS == "linux" {
				// 	if file, err = conn.File(); err == nil {
				// 		idx = uint64(file.Fd())
				// 	}
				// } else {
				// 	idx = server.accepted
				// 	server.accepted++
				// }

				// idx = server.accepted
				server.accepted++

				if err = server.OnNewConn(conn); err == nil {
					client = server.CreateClient(conn, server, server.NewCipher())
					client.start()
					server.addClient(client)
				} else {
					logDebug("[TcpServer %s] init conn error: %v\n", server.tag, err)
				}
			} else {
				conn.Close()
			}
		} else {
			if ne, ok := err.(net.Error); ok && ne.Temporary() {
				if tempDelay == 0 {
					tempDelay = 5 * time.Millisecond
				} else {
					tempDelay *= 2
				}
				if max := 1 * time.Second; tempDelay > max {
					tempDelay = max
				}
				logDebug("[TcpServer %s] Accept error: %v; retrying in %v", server.tag, err, tempDelay)
				time.Sleep(tempDelay)
			} else {
				logDebug("[TcpServer %s] Accept error: %v", server.tag, err)
				if server.onStop != nil {
					server.onStop()
				}
				break
			}
		}
	}

	return err
}

func (server *TcpServer) Start(addr string) error {
	server.Lock()
	running := server.running
	server.running = true
	server.Unlock()

	if !running {
		server.Add(1)

		tcpAddr, err := net.ResolveTCPAddr("tcp", addr)
		if err != nil {
			logFatal("[TcpServer %s] ResolveTCPAddr error: %v", server.tag, err)
			return err
		}

		server.listener, err = net.ListenTCP("tcp", tcpAddr)
		if err != nil {
			logFatal("[TcpServer %s] Listening error: %v", server.tag, err)
			return err
		}

		server.addr = addr
		defer server.listener.Close()

		return server.listenerLoop()
	}
	return fmt.Errorf("server already started")
}

func (server *TcpServer) Stop() {
	server.Lock()
	running := server.running
	server.running = false
	server.Unlock()
	defer handlePanic()

	if !running {
		return
	}

	server.listener.Close()
	server.Done()

	if server.stopTimeout > 0 {
		timer := time.AfterFunc(server.stopTimeout, func() {
			logDebug("[TcpServer %s] Stop Timeout.", server.tag)
			if server.onStop != nil {
				server.onStop()
			}
		})
		defer timer.Stop()
	}

	logDebug("[TcpServer %s] Stop Waiting...", server.tag)

	server.Wait()

	server.stopClients()

	if server.onStopHandler != nil {
		server.onStopHandler(server)
	}
	logDebug("[TcpServer %s] Stop Done.", server.tag)
}

func (server *TcpServer) StopWithTimeout(stopTimeout time.Duration, onStop func()) {
	server.stopTimeout = stopTimeout
	server.onStop = onStop
	server.Stop()
}

func (server *TcpServer) Serve(addr string, stopTimeout time.Duration) {
	safeGo(func() {
		server.Start(addr)
	})

	server.stopTimeout = stopTimeout
	server.onStop = func() {
		os.Exit(0)
	}

	handleSignal(func(sig os.Signal) {
		if sig == syscall.SIGTERM || sig == syscall.SIGINT {
			server.Stop()
			os.Exit(0)
		}
	})
}

func (server *TcpServer) CurrLoad() int64 {
	return atomic.LoadInt64(&server.currLoad)
}

func (server *TcpServer) MaxLoad() int64 {
	return server.maxLoad
}

func (server *TcpServer) SetMaxConcurrent(maxLoad int64) {
	server.maxLoad = maxLoad
}

func (server *TcpServer) AcceptedNum() int64 {
	return server.accepted
}

//handle message by cmd
func (server *TcpServer) HandleServerStop(stopHandler func(server ITcpServer)) {
	server.onStopHandler = stopHandler
}

func (server *TcpServer) EnableBroadcast() {
	server.enableBroad = true
}

func (server *TcpServer) Broadcast(msg IMessage) {
	if !server.enableBroad {
		panic(ErrorBroadcastNotEnabled)
	}
	server.Lock()
	for c, _ := range server.clients {
		c.SendMsg(msg)
	}
	server.Unlock()
}

func (server *TcpServer) BroadcastWithFilter(msg IMessage, filter func(*TcpClient) bool) {
	if !server.enableBroad {
		panic(ErrorBroadcastNotEnabled)
	}
	server.Lock()
	for c, _ := range server.clients {
		if filter(c) {
			c.SendMsg(msg)
		}
	}
	server.Unlock()
}

func NewTcpServer(tag string) ITcpServer {
	server := &TcpServer{
		TcpEngin: TcpEngin{
			clients: map[*TcpClient]struct{}{},
			handlerMap: map[uint32]func(*TcpClient, IMessage){
				CmdSetReaIp: func(client *TcpClient, msg IMessage) {
					ip := msg.Body()
					if len(ip) == 4 {
						client.SetRealIp(fmt.Sprintf("%d.%d.%d.%d", ip[0], ip[1], ip[2], ip[3]))
					} else {
						client.SetRealIp(string(ip))
					}
				},
			},

			sockNoDelay:       DefaultSockNodelay,
			sockKeepAlive:     DefaultSockKeepalive,
			sendQsize:         DefaultSendQSize,
			sockRecvBufLen:    DefaultSockRecvBufLen,
			sockSendBufLen:    DefaultSockSendBufLen,
			sockMaxPackLen:    DefaultSockPackMaxLen,
			sockRecvBlockTime: DefaultSockRecvBlockTime,
			sockSendBlockTime: DefaultSockSendBlockTime,
			sockKeepaliveTime: DefaultSockKeepaliveTime,
		},
		maxLoad: DefaultMaxOnline,
		tag:     tag,
	}

	cipher := NewCipherGzip(0)
	server.HandleNewCipher(func() ICipher {
		return cipher
	})

	server.HandleDisconnected(server.deleClient)

	return server
}
