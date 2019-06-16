package net

import (
	"time"
)

var (
	DefaultSockNodelay   = true
	DefaultSockKeepalive = false

	DefaultSendQSize         = 512
	DefaultSockRecvBufLen    = 1024
	DefaultSockSendBufLen    = 1024
	DefaultSockPackMaxLen    = 1024 * 1024
	DefaultSockLingerSeconds = 0
	DefaultSockKeepaliveTime = time.Second * 60
	DefaultSockRecvBlockTime = time.Second * 65
	DefaultSockSendBlockTime = time.Second * 5

	DefaultSockRpcSendQSize     = 8192
	DefaultSockRpcRecvBlockTime = time.Second * 3600 * 24

	// 默认最大连接数
	DefaultMaxOnline = int64(40960)

	// 默认读超时时间
	DefaultReadTimeout = time.Second * 35

	// 默认写超时时间
	DefaultWriteTimeout = time.Second * 5

	// 默认Shutdown超时时间
	DefaultShutdownTimeout = time.Second * 5

	DefaultReadLimit int64 = 1024 * 16
)
