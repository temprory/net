package net

import (
	// "encoding/binary"
	"github.com/gorilla/websocket"
	"github.com/temprory/log"
	"sync"
	"time"
)

type WSEngine struct {
	sync.Mutex
	sync.WaitGroup

	// 序列化
	Codec ICodec

	// 读超时时间
	ReadTimeout time.Duration

	// 写超时时间
	WriteTimeout time.Duration

	// 读最大包长限制
	ReadLimit int64

	// 发送队列容量
	SendQSize int

	shutdown bool

	//ctypto cipher
	cipher ICipher

	// 消息处理方法
	handlers map[uint32]func(cli *WSClient, msg IMessage)

	//自定义消息处理
	messageHandler func(cli *WSClient, msg IMessage)

	recvHandler func(cli *WSClient) IMessage

	sendHandler func(cli *WSClient, data []byte) error

	sendQueueFullHandler func(cli *WSClient, msg interface{})

	newCipherHandler func() ICipher
}

func (engine *WSEngine) RecvMsg(cli *WSClient) IMessage {
	if engine.recvHandler != nil {
		return engine.recvHandler(cli)
	}

	var err error
	var data []byte

	if cli.ReadTimeout > 0 {
		err = cli.Conn.SetReadDeadline(time.Now().Add(cli.ReadTimeout))
		if err != nil {
			log.Debug("Websocket SetReadDeadline failed: %v", err)
			return nil
		}
	}

	_, data, err = cli.Conn.ReadMessage()
	if err != nil {
		log.Debug("Websocket ReadIMessage failed: %v", err)
		return nil
	}

	msg := &Message{
		rawData: data,
		data:    nil,
	}

	if _, err = msg.Decrypt(cli.RecvSeq(), cli.RecvKey(), cli.Cipher()); err != nil {
		logDebug("%s RecvMsg Decrypt Err: %v", cli.Conn.RemoteAddr().String(), err)
		return nil
	}

	return msg
}

func (engine *WSEngine) Send(cli *WSClient, data []byte) error {
	if engine.sendHandler != nil {
		return engine.sendHandler(cli, data)
	}

	var err error

	if cli.WriteTimeout > 0 {
		err = cli.Conn.SetWriteDeadline(time.Now().Add(engine.WriteTimeout))
		if err != nil {
			logDebug("%s Send SetReadDeadline Err: %v", cli.Conn.RemoteAddr().String(), err)
			cli.Stop()
			return err
		}
	}

	err = cli.Conn.WriteMessage(websocket.TextMessage, data)
	if err != nil {
		logDebug("%s Send Write Err: %v", cli.Conn.RemoteAddr().String(), err)
		cli.Stop()
	}

	return err
}

func (engine *WSEngine) HandleMessage(h func(cli *WSClient, msg IMessage)) {
	engine.messageHandler = h
}

func (engine *WSEngine) Handle(cmd uint32, h func(cli *WSClient, msg IMessage)) {
	if _, ok := engine.handlers[cmd]; ok {
		log.Panic("Websocket Handle failed, cmd %v already exist", cmd)
	}
	engine.handlers[cmd] = h
}

//setting message router
func (engine *WSEngine) HandleRecv(recver func(cli *WSClient) IMessage) {
	engine.recvHandler = recver
}

func (engine *WSEngine) HandleSend(sender func(cli *WSClient, data []byte) error) {
	engine.sendHandler = sender
}

func (engine *WSEngine) OnSendQueueFull(cli *WSClient, msg interface{}) {
	if engine.sendQueueFullHandler != nil {
		engine.sendQueueFullHandler(cli, msg)
	}
}

func (engine *WSEngine) HandleSendQueueFull(h func(cli *WSClient, msg interface{})) {
	engine.sendQueueFullHandler = h
}

func (engine *WSEngine) NewCipher() ICipher {
	if engine.newCipherHandler != nil {
		return engine.newCipherHandler()
	}
	return nil
}

func (engine *WSEngine) HandleNewCipher(newCipher func() ICipher) {
	engine.newCipherHandler = newCipher
}

// 消息处理
func (engine *WSEngine) onMessage(cli *WSClient, msg IMessage) {
	if engine.shutdown {
		return
	}
	engine.Add(1)
	defer engine.Done()
	defer handlePanic()

	if engine.messageHandler != nil {
		engine.messageHandler(cli, msg)
		return
	}

	if h, ok := engine.handlers[msg.Cmd()]; ok {
		h(cli, msg)
	} else {
		log.Debug("Websocket no handler for cmd: %v", msg.Cmd())
	}
}

func NewWebsocketEngine() *WSEngine {
	engine := &WSEngine{
		Codec:        DefaultCodec,
		ReadTimeout:  DefaultReadTimeout,
		WriteTimeout: DefaultWriteTimeout,
		ReadLimit:    DefaultReadLimit,
		SendQSize:    DefaultSendQSize,
		shutdown:     false,
		handlers:     map[uint32]func(cli *WSClient, msg IMessage){},
	}

	cipher := NewCipherGzip(DefaultThreshold)
	engine.HandleNewCipher(func() ICipher {
		return cipher
	})

	return engine
}
