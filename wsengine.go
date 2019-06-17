package net

import (
	// "encoding/binary"
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
	handlers map[uint32]func(cli *WSClient, msg *Message)

	//自定义消息处理
	messageHandler func(cli *WSClient, msg *Message)

	sendQueueFullHandler func(cli *WSClient, msg interface{})

	newCipherHandler func() ICipher
}

func (engine *WSEngine) HandleMessage(h func(cli *WSClient, msg *Message)) {
	engine.messageHandler = h
}

func (engine *WSEngine) Handle(cmd uint32, h func(cli *WSClient, msg *Message)) {
	if _, ok := engine.handlers[cmd]; ok {
		log.Panic("Websocket Handle failed, cmd %v already exist", cmd)
	}
	engine.handlers[cmd] = h
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
func (engine *WSEngine) onMessage(cli *WSClient, msg *Message) {
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
		handlers:     map[uint32]func(cli *WSClient, msg *Message){},
	}

	cipher := NewCipherGzip(DefaultThreshold)
	engine.HandleNewCipher(func() ICipher {
		return cipher
	})

	return engine
}
