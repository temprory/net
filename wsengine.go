package net

import (
	"encoding/binary"
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
	handlers map[uint32]func(cli *WSClient, cmd uint32, data []byte)

	//自定义消息处理
	messageHandler func(cli *WSClient, data []byte)

	sendQueueFullHandler func(cli *WSClient, msg interface{})

	newCipherHandler func() ICipher
}

func (engine *WSEngine) HandleMessage(h func(cli *WSClient, data []byte)) {
	engine.messageHandler = h
}

func (engine *WSEngine) Handle(cmd uint32, h func(cli *WSClient, cmd uint32, data []byte)) {
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
func (engine *WSEngine) onMessage(cli *WSClient, data []byte) {
	if engine.shutdown {
		return
	}
	engine.Add(1)
	defer engine.Done()
	defer handlePanic()

	if engine.messageHandler != nil {
		engine.messageHandler(cli, data)
		return
	}

	if len(data) < 4 {
		log.Debug("Websocket invalid data len: %v, shutdown client", len(data))
		cli.Stop()
		return
	}

	cmd := binary.LittleEndian.Uint32(data[:4])

	if h, ok := engine.handlers[cmd]; ok {
		h(cli, cmd, data[4:])
	} else {
		log.Debug("Websocket no handler for cmd: %v", cmd)
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
		handlers:     map[uint32]func(cli *WSClient, cmd uint32, data []byte){},
	}

	cipher := NewCipherGzip(0)
	engine.HandleNewCipher(func() ICipher {
		return cipher
	})

	return engine
}
