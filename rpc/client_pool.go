package rpc

import (
	"fmt"
	"sync/atomic"
	"time"
)

type ClientPool struct {
	idx     int64
	size    uint64
	clients []*Client
}

func (pool *ClientPool) Call(method string, req interface{}, rsp interface{}, timeout time.Duration) error {
	idx := uint64(atomic.AddInt64(&pool.idx, 1)) % pool.size
	return pool.clients[idx].Call(method, req, rsp, timeout)
}

func NewClientPool(addr string, size int) *ClientPool {
	if size <= 0 {
		panic(fmt.Errorf("NewClientPool failed: invalid size %v", size))
	}
	clients := make([]*Client, size)
	for i := 0; i < size; i++ {
		clients[i] = NewClient(addr)
	}
	return &ClientPool{
		idx:     0,
		size:    uint64(size),
		clients: clients,
	}
}
