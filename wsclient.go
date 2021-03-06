package net

import (
	// "encoding/binary"
	// "flag"
	// "html/template"
	// "net"
	"crypto/tls"
	"strconv"
	"strings"
	"sync"
	"time"

	//"compress/flate"
	"github.com/gorilla/websocket"
	//"github.com/temprory/log"
	//"github.com/valyala/fasthttp"
)

type WSClient struct {
	*WSEngine

	sync.RWMutex
	Conn *websocket.Conn

	User interface{}

	realIp  string
	chSend  chan wsAsyncMessage
	running bool

	cipher ICipher

	//recv packet sequence
	recvSeq int64

	//send packet sequence
	sendSeq int64

	//pre recv packet key
	recvKey uint32

	//pre send packet key
	sendKey uint32

	userdata interface{}

	onCloseMap map[interface{}]func(*WSClient)
}

// type wsAsyncMessage struct {
// 	cmd  uint32
// 	data interface{}
// 	cb   func(*WSClient, error)
// }

func (cli *WSClient) reader() {
	defer handlePanic()
	defer cli.Stop()

	var imsg IMessage
	for {
		if imsg = cli.WSEngine.RecvMsg(cli); imsg == nil {
			break
		}
		cli.recvSeq++
		cli.WSEngine.onMessage(cli, imsg)
	}
}

func (cli *WSClient) writer() {
	defer cli.Stop()
	defer handlePanic()

	var err error
	for msg := range cli.chSend {
		err = cli.WSEngine.Send(cli, msg.data)
		if msg.cb != nil {
			msg.cb(cli, err)
		}
		if err != nil {
			break
		}

		cli.sendSeq++
	}
}

func (cli *WSClient) Keepalive(interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	msg := PingMsg()
	for {
		<-ticker.C

		if !cli.running {
			return
		}

		cli.SendMsg(msg)
	}
}

func (cli *WSClient) RecvSeq() int64 {
	return cli.recvSeq
}

func (cli *WSClient) SendSeq() int64 {
	return cli.sendSeq
}

func (cli *WSClient) RecvKey() uint32 {
	return cli.recvKey
}

func (cli *WSClient) SendKey() uint32 {
	return cli.sendKey
}

func (cli *WSClient) Cipher() ICipher {
	return cli.cipher
}

func (cli *WSClient) SetCipher(cipher ICipher) {
	cli.cipher = cipher
}

func (cli *WSClient) UserData() interface{} {
	return cli.userdata
}

func (cli *WSClient) SetUserData(data interface{}) {
	cli.userdata = data
}

func (cli *WSClient) Ip() string {
	if cli.realIp != "" {
		return cli.realIp
	}
	if cli.Conn != nil {
		addr := cli.Conn.RemoteAddr().String()
		if pos := strings.LastIndex(addr, ":"); pos > 0 {
			return addr[:pos]
		}
	}
	return "0.0.0.0"
}

func (cli *WSClient) Port() int {
	if cli.Conn != nil {
		addr := cli.Conn.RemoteAddr().String()
		if pos := strings.LastIndex(addr, ":"); pos > 0 {
			if port, err := strconv.Atoi(addr[pos+1:]); err == nil {
				return port
			}
		}
	}
	return 0
}

func (cli *WSClient) SetRealIp(ip string) {
	cli.realIp = ip
}

func (cli *WSClient) Bind(data []byte, v interface{}) error {
	if cli.Codec == nil {
		return ErrClientWithoutCodec
	}
	return cli.Codec.Unmarshal(data, v)
}

func (cli *WSClient) SendMsg(msg IMessage) error {
	var err error = nil
	cli.Lock()
	if cli.running {
		select {
		case cli.chSend <- wsAsyncMessage{msg.Encrypt(cli.SendSeq(), cli.SendKey(), cli.cipher), nil}:
			cli.Unlock()
		default:
			cli.Unlock()
			cli.OnSendQueueFull(cli, msg)
			err = ErrWSClientSendQueueIsFull
		}
	} else {
		cli.Unlock()
		err = ErrWSClientIsStopped
	}
	if err != nil {
		logDebug("[Websocket] SendMsg -> %v failed: %v", cli.Ip(), err)
	}

	return err
}

func (cli *WSClient) SendMsgWithCallback(msg IMessage, cb func(*WSClient, error)) error {
	var err error = nil
	cli.Lock()
	if cli.running {
		select {
		case cli.chSend <- wsAsyncMessage{msg.Encrypt(cli.SendSeq(), cli.SendKey(), cli.cipher), cb}:
			cli.Unlock()
		default:
			cli.Unlock()
			cli.OnSendQueueFull(cli, msg)
			err = ErrTcpClientSendQueueIsFull
		}
	} else {
		cli.Unlock()
		err = ErrWSClientIsStopped
	}
	if err != nil {
		logDebug("SendMsgWithCallback -> %v failed: %v", cli.Ip(), err)
	}

	return err
}

func (cli *WSClient) SendData(data []byte) error {
	var err error = nil
	cli.Lock()
	if cli.running {
		select {
		case cli.chSend <- wsAsyncMessage{data, nil}:
			cli.Unlock()
		default:
			cli.Unlock()
			cli.OnSendQueueFull(cli, data)
			err = ErrTcpClientSendQueueIsFull
		}
	} else {
		cli.Unlock()
		err = ErrWSClientIsStopped
	}
	if err != nil {
		logDebug("SendData -> %v failed: %v", cli.Ip(), err)
	}

	return err
}

func (cli *WSClient) SendDataWithCallback(data []byte, cb func(*WSClient, error)) error {
	var err error = nil
	cli.Lock()
	if cli.running {
		select {
		case cli.chSend <- wsAsyncMessage{data, cb}:
			cli.Unlock()
		default:
			cli.Unlock()
			cli.OnSendQueueFull(cli, data)
			err = ErrWSClientSendQueueIsFull
		}
	} else {
		cli.Unlock()
		err = ErrWSClientIsStopped
	}
	if err != nil {
		logDebug("SendDataWithCallback -> %v failed: %v", cli.Ip(), err)
	}

	return err
}

func (cli *WSClient) Stop() {
	cli.Lock()
	running := cli.running
	if running {
		cli.running = false
		cli.Conn.Close()
		close(cli.chSend)
	}
	cli.Unlock()
	if running {
		cli.RLock()
		for _, cb := range cli.onCloseMap {
			cb(cli)
		}
		cli.RUnlock()
	}
}

func (cli *WSClient) OnClose(tag interface{}, cb func(client *WSClient)) {
	cli.Lock()
	cli.onCloseMap[tag] = cb
	cli.Unlock()
}

func (cli *WSClient) CancelOnClose(tag interface{}) {
	cli.Lock()
	delete(cli.onCloseMap, tag)
	cli.Unlock()
}

// func (cli *WSClient) Handle(cmd uint32, h func(cli *WSClient, cmd uint32, data []byte)) {
// 	cli.Handle(cmd, h)
// }

func newClient(conn *websocket.Conn, engine *WSEngine) *WSClient {
	sendQSize := DefaultSendQSize
	if engine != nil && engine.SendQSize > 0 {
		sendQSize = engine.SendQSize
	}

	cipher := engine.NewCipher()
	cli := &WSClient{
		WSEngine:   engine,
		Conn:       conn,
		chSend:     make(chan wsAsyncMessage, sendQSize),
		running:    true,
		cipher:     cipher,
		onCloseMap: map[interface{}]func(*WSClient){},
	}

	addr := conn.RemoteAddr().String()
	if pos := strings.LastIndex(addr, ":"); pos > 0 {
		cli.realIp = addr[:pos]
	}

	return cli
}

func NewWebsocketClient(addr string) (*WSClient, error) {
	dialer := &websocket.Dialer{}
	dialer.TLSClientConfig = &tls.Config{}
	conn, _, err := dialer.Dial(addr, nil)

	// conn.EnableWriteCompression(true)
	// conn.SetCompressionLevel(flate.BestCompression)

	if err != nil {
		return nil, err
	}

	cli := newClient(conn, NewWebsocketEngine())

	// go cli.readloop()

	// go cli.writeloop()

	safeGo(cli.reader)
	safeGo(cli.writer)

	return cli, nil
}

func NewWebsocketTLSClient(addr string) (*WSClient, error) {
	dialer := &websocket.Dialer{}
	dialer.TLSClientConfig = &tls.Config{}
	// cert, err := tls.LoadX509KeyPair(certfile, keyfile)
	// if err != nil {
	// 	return nil, err
	// }
	// dialer.TLSClientConfig.Certificates = append(dialer.TLSClientConfig.Certificates, cert)
	dialer.TLSClientConfig.InsecureSkipVerify = true
	conn, _, err := dialer.Dial(addr, nil)

	// conn.EnableWriteCompression(true)
	// conn.SetCompressionLevel(flate.BestCompression)

	if err != nil {
		return nil, err
	}

	cli := newClient(conn, NewWebsocketEngine())

	safeGo(cli.reader)
	safeGo(cli.writer)

	return cli, nil
}
