package net

import (
	"net"
	"strconv"
	"strings"
	"sync"
	//"sync/atomic"
	"time"
)

type ITcpClient interface {
	Conn() *net.TCPConn

	Ip() string
	SetRealIp(string)

	Port() int

	Lock()
	Unlock()

	//IsRunning() bool

	OnClose(tag interface{}, cb func(ITcpClient))
	CancelOnClose(tag interface{})

	SendMsg(msg IMessage) error
	SendMsgWithCallback(msg IMessage, cb func(client ITcpClient, err error)) error
	SendData(data []byte) error
	SendDataWithCallback(data []byte, cb func(client ITcpClient, err error)) error

	SendMsgSync(msg IMessage) error
	SendMsgSyncWithoutLock(msg IMessage) error
	SendDataSync(data []byte) error
	SendDataSyncWithoutLock(data []byte) error

	RecvSeq() int64
	SendSeq() int64

	RecvKey() uint32
	SendKey() uint32

	Cipher() ICipher
	SetCipher(cipher ICipher)

	UserData() interface{}
	SetUserData(interface{})

	Keepalive(interval time.Duration)
	Stop() error
	Shutdown() error

	start()
}

type TcpClient struct {
	sync.Mutex

	//tcp connection
	conn *net.TCPConn

	//tcp server parent
	parent ITcpEngin

	//recv packet sequence
	recvSeq int64

	//send packet sequence
	sendSeq int64

	//pre recv packet key
	recvKey uint32

	//pre send packet key
	sendKey uint32

	//ctypto cipher
	cipher ICipher

	//chan used for stop
	//chStop chan struct{}

	//chan for message send queue
	chSend chan asyncMessage

	//client close callbacks
	onCloseMap map[interface{}]func(ITcpClient)

	userdata interface{}

	realIp string

	running bool

	shutdown bool
}

func (client *TcpClient) Conn() *net.TCPConn {
	return client.conn
}

func (client *TcpClient) Ip() string {
	if client.realIp != "" {
		return client.realIp
	}
	if client.conn != nil {
		addr := client.conn.RemoteAddr().String()
		if pos := strings.LastIndex(addr, ":"); pos > 0 {
			return addr[:pos]
		}
	}
	return "0.0.0.0"
}

func (client *TcpClient) Port() int {
	if client.conn != nil {
		addr := client.conn.RemoteAddr().String()
		if pos := strings.LastIndex(addr, ":"); pos > 0 {
			if port, err := strconv.Atoi(addr[pos+1:]); err == nil {
				return port
			}
		}
	}
	return 0
}

func (client *TcpClient) SetRealIp(ip string) {
	client.realIp = ip
}

func (client *TcpClient) IsRunning() bool {
	return client.running
}

func (client *TcpClient) OnClose(tag interface{}, cb func(client ITcpClient)) {
	client.Lock()
	if client.running {
		client.onCloseMap[tag] = cb
	}
	client.Unlock()
}

func (client *TcpClient) CancelOnClose(tag interface{}) {
	client.Lock()
	if client.running {
		delete(client.onCloseMap, tag)
	}
	client.Unlock()
}

func (client *TcpClient) SendMsg(msg IMessage) error {
	var err error = nil
	client.Lock()
	if client.running {
		select {
		case client.chSend <- asyncMessage{msg.Encrypt(client.SendSeq(), client.SendKey(), client.cipher), nil}:
			client.Unlock()
		default:
			client.Unlock()
			client.parent.OnSendQueueFull(client, msg)
			err = ErrTcpClientSendQueueIsFull
		}
	} else {
		client.Unlock()
		err = ErrTcpClientIsStopped
	}
	if err != nil {
		logDebug("SendMsg -> %v failed: %v", client.Ip(), err)
	}

	return err
}

func (client *TcpClient) SendMsgWithCallback(msg IMessage, cb func(ITcpClient, error)) error {
	var err error = nil
	client.Lock()
	if client.running {
		select {
		case client.chSend <- asyncMessage{msg.Encrypt(client.SendSeq(), client.SendKey(), client.cipher), cb}:
			client.Unlock()
		default:
			client.Unlock()
			client.parent.OnSendQueueFull(client, msg)
			err = ErrTcpClientSendQueueIsFull
		}
	} else {
		client.Unlock()
		err = ErrTcpClientIsStopped
	}
	if err != nil {
		logDebug("SendMsgWithCallback -> %v failed: %v", client.Ip(), err)
	}

	return err
}

func (client *TcpClient) SendData(data []byte) error {
	var err error = nil
	client.Lock()
	if client.running {
		select {
		case client.chSend <- asyncMessage{data, nil}:
			client.Unlock()
		default:
			client.Unlock()
			client.parent.OnSendQueueFull(client, data)
			err = ErrTcpClientSendQueueIsFull
		}
	} else {
		client.Unlock()
		err = ErrTcpClientIsStopped
	}
	if err != nil {
		logDebug("SendData -> %v failed: %v", client.Ip(), err)
	}

	return err
}

func (client *TcpClient) SendDataWithCallback(data []byte, cb func(ITcpClient, error)) error {
	var err error = nil
	client.Lock()
	if client.running {
		select {
		case client.chSend <- asyncMessage{data, cb}:
			client.Unlock()
		default:
			client.Unlock()
			client.parent.OnSendQueueFull(client, data)
			err = ErrTcpClientSendQueueIsFull
		}
	} else {
		client.Unlock()
		err = ErrTcpClientIsStopped
	}
	if err != nil {
		logDebug("SendDataWithCallback -> %v failed: %v", client.Ip(), err)
	}

	return err
}

func (client *TcpClient) SendMsgSync(msg IMessage) error {
	defer handlePanic()
	client.Lock()
	if client.running {
		client.Unlock()
		return client.parent.Send(client, msg.Encrypt(client.SendSeq(), client.SendKey(), client.cipher))
	}
	client.Unlock()
	return ErrTcpClientIsStopped
}

func (client *TcpClient) SendMsgSyncWithoutLock(msg IMessage) error {
	defer handlePanic()
	return client.parent.Send(client, msg.Encrypt(client.SendSeq(), client.SendKey(), client.cipher))
}

func (client *TcpClient) SendDataSync(data []byte) error {
	defer handlePanic()
	client.Lock()
	if client.running {
		client.Unlock()
		return client.parent.Send(client, data)
	}
	client.Unlock()
	return ErrTcpClientIsStopped
}

func (client *TcpClient) SendDataSyncWithoutLock(data []byte) error {
	defer handlePanic()
	return client.parent.Send(client, data)
}

func (client *TcpClient) RecvSeq() int64 {
	return client.recvSeq
}

func (client *TcpClient) SendSeq() int64 {
	return client.sendSeq
}

func (client *TcpClient) RecvKey() uint32 {
	return client.recvKey
}

func (client *TcpClient) SendKey() uint32 {
	return client.sendKey
}

func (client *TcpClient) Cipher() ICipher {
	return client.cipher
}

func (client *TcpClient) SetCipher(cipher ICipher) {
	client.cipher = cipher
}

func (client *TcpClient) UserData() interface{} {
	return client.userdata
}

func (client *TcpClient) SetUserData(data interface{}) {
	client.userdata = data
}

func (client *TcpClient) start() {
	safeGo(client.reader)
	safeGo(client.writer)
}

func (client *TcpClient) Keepalive(interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	msg := PingMsg()
	for {
		<-ticker.C

		if client.shutdown {
			return
		}
		client.SendMsg(msg)
	}
}

func (client *TcpClient) restart(conn *net.TCPConn) {
	client.Lock()
	defer client.Unlock()
	if !client.running {
		client.running = true

		client.conn = conn
		if client.cipher != nil {
			client.cipher.Init()
		}
		sendQsize := client.parent.SendQueueSize()
		if sendQsize <= 0 {
			sendQsize = _conf_sock_send_q_size
		}
		client.chSend = make(chan asyncMessage, sendQsize)

		safeGo(client.writer)
		safeGo(client.reader)
	}
}

//only called once when reader exit
func (client *TcpClient) stop() {
	defer handlePanic()

	client.Lock()
	client.running = false
	client.Unlock()

	close(client.chSend)

	client.conn.CloseRead()
	client.conn.CloseWrite()
	client.conn.Close()

	for _, cb := range client.onCloseMap {
		cb(client)
	}

	client.parent.OnDisconnected(client)
}

//for normal client created by server
func (client *TcpClient) Stop() error {
	defer handlePanic()
	client.Lock()
	running := client.running
	client.running = false
	client.Unlock()
	if running {
		if client.conn != nil {
			err := client.conn.CloseRead()
			if err != nil {
				return err
			}
			return client.conn.CloseWrite()
			//return client.conn.Close()
		}
	}
	return ErrTcpClientIsStopped
}

//for rpc client with auto reconnect
func (client *TcpClient) Shutdown() error {
	client.Lock()
	client.shutdown = true
	client.Unlock()
	return client.Stop()
}

/******************************************************/
func (client *TcpClient) send(amsg *asyncMessage) error {
	defer handlePanic()
	return client.parent.Send(client, amsg.data)
}

func (client *TcpClient) writer() {
	var err error = nil
	for asyncMsg := range client.chSend {
		err = client.send(&asyncMsg)
		if asyncMsg.cb != nil {
			asyncMsg.cb(client, err)
		}
		if err != nil {
			client.Stop()
			//break
		}
		client.sendSeq++
	}
}

func (client *TcpClient) reader() {
	defer client.stop()
	var imsg IMessage
	for {
		if imsg = client.parent.RecvMsg(client); imsg == nil {
			break
		}
		client.recvSeq++
		client.parent.OnMessage(client, imsg)
	}
}

func createTcpClient(conn *net.TCPConn, parent ITcpEngin, cipher ICipher) *TcpClient {
	sendQsize := parent.SendQueueSize()
	if sendQsize <= 0 {
		sendQsize = _conf_sock_send_q_size
	}

	conn.SetNoDelay(parent.SockNoDelay())
	conn.SetKeepAlive(parent.SockKeepAlive())
	if parent.SockKeepAlive() {
		conn.SetKeepAlivePeriod(parent.SockKeepaliveTime())
	}
	conn.SetReadBuffer(parent.SockRecvBufLen())
	conn.SetWriteBuffer(parent.SockSendBufLen())

	return &TcpClient{
		conn:       conn,
		parent:     parent,
		cipher:     cipher,
		chSend:     make(chan asyncMessage, sendQsize),
		onCloseMap: map[interface{}]func(ITcpClient){},
		running:    true,
	}
}

func NewTcpClient(addr string, parent ITcpEngin, cipher ICipher, autoReconn bool, onConnected func(ITcpClient)) (*TcpClient, error) {
	tcpAddr, err := net.ResolveTCPAddr("tcp", addr)
	if err != nil {
		logDebug("NewTcpClient failed: ", err)
		return nil, err
	}
	conn, err := net.DialTCP("tcp", nil, tcpAddr)
	if err != nil {
		logDebug("NewTcpClient failed: %v", err)
		return nil, err
	}

	client := createTcpClient(conn, parent, cipher)
	client.start()

	if autoReconn {
		client.OnClose("reconn", func(ITcpClient) {
			safeGo(func() {
				times := 0
				tempDelay := time.Second / 10
				for !client.shutdown {
					times++
					time.Sleep(tempDelay)
					if conn, err := net.DialTCP("tcp", nil, tcpAddr); err == nil {
						client.Lock()
						defer client.Unlock()
						if !client.shutdown {
							logDebug("TcpClient auto reconnect to %v %d success", addr, times)
							client.recvSeq = 0
							client.sendSeq = 0
							safeGo(func() {
								client.restart(conn)
								if onConnected != nil {
									onConnected(client)
								}
							})
						} else {
							conn.Close()
						}
						return
					} else {
						logDebug("TcpClient auto reconnect to %v %d failed: %v", addr, times, err)
					}
					tempDelay *= 2
					if tempDelay > time.Second*2 {
						tempDelay = time.Second * 2
					}
				}
			})
		})
	}
	if onConnected != nil {
		safeGo(func() {
			onConnected(client)
		})
	}

	return client, nil
}
