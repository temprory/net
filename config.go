package net

import (
	"time"
)

var (
	_conf_sock_nodelay   = true
	_conf_sock_keepalive = false

	_conf_gzip_enabled   = true
	_conf_auto_gzip_size = 1024 - 16

	_conf_sock_send_q_size     = 256
	_conf_sock_recv_buf_len    = 1024
	_conf_sock_send_buf_len    = 1024
	_conf_sock_pack_max_len    = 1024 * 1024
	_conf_sock_linger_seconds  = 0
	_conf_sock_keepalive_time  = time.Second * 60
	_conf_sock_recv_block_time = time.Second * 65
	_conf_sock_send_block_time = time.Second * 5

	_conf_sock_rpc_send_q_size     = 4096
	_conf_sock_rpc_recv_block_time = time.Second * 3600 * 24
)
