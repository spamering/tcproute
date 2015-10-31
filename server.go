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

type Server struct {
	Addr           string        // TCP address to listen on, ":http" if empty
	Handlers       []Newer       // handler to invoke, http.DefaultServeMux if nil
	ReadTimeout    time.Duration // maximum duration before timing out read of the request
	WriteTimeout   time.Duration // maximum duration before timing out write of the response
	MaxHeaderBytes int           // maximum size of request headers, DefaultMaxHeaderBytes if 0
	ln             net.Listener
}
type liveSwitchReader struct {
	sync.Mutex
	r io.Reader
}

type conn struct {
	Server     *Server
	Rwc        net.Conn          // 原始链接
	remoteAddr net.Addr          //远端地址
	w          io.Writer         //写
	sr         io.Reader
	rb         *bytes.Buffer     // 重置流缓冲区
	tr         *io.Reader        //TeeReader  读，同时写重置缓冲区
	mr         *io.Reader        //MultiReader 重置缓冲区+标准读
	lr         *io.LimitedReader //
	br         *bufio.Reader     // 缓冲读
	bw         *bufio.Writer     //缓冲写

	buf        *bufio.ReadWriter // buffered(lr,rwc), reading from bufio->limitReader->sr->rwc


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
		c, err := srv.newConn(rw)
		if err != nil {
			continue
		}
		go c.handle()
	}
}

func (srv *Server) newConn(rw net.Conn) (c *Conn, err error) {


	return c, nil
}

// server 单个连接处理函数
func (c *Conn) handle() {
	server := c.Server
	handlers := server.Handlers

	for _, v := range handlers {
		h, e := v.New(c)
		if e != nil {
			continue
		}
		glog.Infof("[%v]收到 %v 的请求。", h.String(), c.remoteAddr.String())
		h.Handle()
	}
}
