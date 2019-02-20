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

	ErrorReservedCmdPing           = fmt.Errorf("cmd %d is reserved for ping, plz use other number", CmdPing)
	ErrorReservedCmdSetRealip      = fmt.Errorf("cmd %d is reserved for set client's real ip, plz use other number", CmdSetReaIp)
	ErrorReservedCmdRpcMethod      = fmt.Errorf("cmd %d is reserved for rpc method, plz use other number", CmdRpcMethod)
	ErrorReservedCmdRpcMethodError = fmt.Errorf("cmd %d is reserved for rpc method error, plz use other number", CmdRpcMethodError)
)
