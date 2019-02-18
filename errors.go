package net

import (
	"errors"
	"fmt"
)

var (
	ErrTcpClientIsStopped       = errors.New("tcp client is stopped")
	ErrTcpClientSendQueueIsFull = errors.New("tcp client's send queue is full")

	ErrRpcClientIsDisconnected  = errors.New("rpc client disconnected")
	ErrRpcClientSendQueueIsFull = errors.New("rpc client's send queue is full")
	ErrRpcCallTimeout           = errors.New("rpc call timeout")
	ErrRpcCallClientError       = errors.New("rpc client error")

	ErrorBroadcastNotEnabled = errors.New("broadcast not enabled")

	ErrorReservedCmdPing      = fmt.Errorf("cmd %d is reserved for ping", CmdSetReaIp)
	ErrorReservedCmdSetRealip = fmt.Errorf("cmd %d is reserved for set client's real ip", CmdSetReaIp)
)
