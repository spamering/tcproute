package main

import (
	"time"
	"net"
	"github.com/golang/glog"
	"io"
	"bufio"
	"sync"
	"bytes"
)

//
const (
// 尝试创建处理器时的conn.read 的timeout
// 实际是一次性读取数据，所以这个超时指的是客户端必须10秒内发出第一和数据
	handler_new_timeout = 10 * time.Second

// 默认一个连接的总处理时间
	handler_base_timeout = 10 * time.Minute
)

type Server struct {
	Addr   string       // TCP address to listen on, ":http" if empty
	hNewer HandlerNewer //
						//	ReadTimeout    time.Duration // maximum duration before timing out read of the request
						//	WriteTimeout   time.Duration // maximum duration before timing out write of the response
						//	MaxHeaderBytes int           // maximum size of request headers, DefaultMaxHeaderBytes if 0
	ln     net.Listener
}

func (srv *Server) ListAndServe() {
	if srv.Addr == "" {
		srv.Addr = ":7070"
	}

	ln, err := net.Listen("tcp", srv.Addr)
	if err != nil {
		panic(err)
	}
	srv.ln = ln

	return srv.Server()
}


func (srv *Server) Server() {
	ln := &srv.ln
	defer ln.Close()
	var tempDelay time.Duration
	for {
		rw, e := ln.Accept()
		if e != nil {
			if ne, ok := e.(net.Error); ok && ne.Temporary() {
				if tempDelay == 0 {
					tempDelay = 5 * time.Millisecond
				} else {
					tempDelay *= 2
				}
				if max := 1 * time.Second; tempDelay > max {
					tempDelay = max
				}
				glog.Warning("Accept error: %v; retrying in %v", e, tempDelay)
				time.Sleep(tempDelay)
				continue
			}
			return e
		}
		tempDelay = 0

		go srv.handlerConn(rw)
	}
}

func (srv *Server) handlerConn(conn net.Conn) {
	defer func() {
		if err := recover(); err != nil {
			glog.Error("work failed:", err)
		}
	}()
	// 是这里调用关闭还是 Handler() 负责？
	defer conn.Close()

	if tcpConn, ok := conn.(*net.TCPConn); ok == true {
		// 设置关闭连接时最多等待多少秒
		tcpConn.SetLinger(5)
	}
	conn.SetDeadline(time.Now().Add(handler_new_timeout))

	h, _, err := srv.hNewer.New(conn)
	if h == nil {
		glog.Warning("无法识别请求的协议类型，远端地址：%v，近端地址：%v，详细错误：%v", conn.RemoteAddr(), conn.LocalAddr(), err)
		return
	}

	conn.SetDeadline(time.Now().Add(handler_base_timeout))
	h.Handle()
}
