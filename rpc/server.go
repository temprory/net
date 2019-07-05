package rpc

import (
	"fmt"
	"github.com/valyala/fastrpc"
	"github.com/valyala/fastrpc/tlv"
	"net"
	"time"
)

const (
	cmd_msg = iota
	cmd_error
)

type Server struct {
	s      *fastrpc.Server
	ln     net.Listener
	chStop chan error
	routes map[string]func(*Context)
}

func (s *Server) newHandlerCtx() fastrpc.HandlerCtx {
	return &tlv.RequestCtx{
		ConcurrencyLimitErrorHandler: func(ctx *tlv.RequestCtx, concurrency int) {
			ctx.Response.SwapValue([]byte("too many requests"))
		},
	}
}

func (s *Server) Printf(format string, args ...interface{}) {
	fmt.Printf(format, args...)
}

func (s *Server) ListenAndServe(addr string) error {
	s.s = &fastrpc.Server{
		NewHandlerCtx: func() fastrpc.HandlerCtx {
			return &tlv.RequestCtx{
				ConcurrencyLimitErrorHandler: func(ctx *tlv.RequestCtx, concurrency int) {
					ctx.Response.SwapValue([]byte("too many requests"))
				},
			}
		},
		Handler: s.onIMessage,
		Logger:  s,
	}

	ln, err := net.Listen("tcp", addr)
	if err != nil {
		panic(fmt.Errorf("Server Listen failed: %v", err))
	}

	s.ln = ln
	s.chStop = make(chan error, 1)
	fmt.Println("rpc server running on:", addr)
	defer fmt.Println("rpc server stopped.")
	err = s.s.Serve(ln)
	s.chStop <- err
	return err
}

func (s *Server) Shutdown() error {
	s.ln.Close()
	select {
	case err := <-s.chStop:
		if err != nil {
			return fmt.Errorf("unexpected error: %s", err)
		}
	case <-time.After(time.Second):
		return fmt.Errorf("timeout")
	}
	return nil
}

func (s *Server) Handle(method string, handler func(*Context)) {
	if len(s.routes) == 0 {
		s.routes = map[string]func(*Context){}
	}
	if _, ok := s.routes[method]; ok {
		panic(fmt.Errorf("handler exist for method %v ", method))
	}
	s.routes[method] = handler
}

func (s *Server) onIMessage(ctxv fastrpc.HandlerCtx) fastrpc.HandlerCtx {
	safe(func() {
		ctx := ctxv.(*tlv.RequestCtx)
		data := ctx.Request.Value()
		if len(data) == 0 {
			ctx.Write(append([]byte{cmd_error}, []byte("invalid call")...))
			return
		}
		methodLen := int(data[0])
		if methodLen <= 0 || methodLen >= 128 || len(data)-1 < methodLen {
			ctx.Write(append([]byte{cmd_error}, []byte("invalid method")...))
			return
		}
		method := string(data[1 : 1+methodLen])
		if handler, ok := s.routes[method]; ok {
			handler(&Context{ctx: ctx, data: data[1+methodLen:]})
		} else {
			ctx.Write(append([]byte{cmd_error}, []byte("method not found: "+method)...))
		}
	})
	return ctxv
}
