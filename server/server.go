package server

import (
	"time"
	"net"
	"github.com/gamexg/TcpRoute2/handler"
	"github.com/golang/glog"
)

type Server struct {
	Addr           string            // TCP address to listen on, ":http" if empty
	Handler        []handler.Handler // handler to invoke, http.DefaultServeMux if nil
	ReadTimeout    time.Duration     // maximum duration before timing out read of the request
	WriteTimeout   time.Duration     // maximum duration before timing out write of the response
	MaxHeaderBytes int               // maximum size of request headers, DefaultMaxHeaderBytes if 0
	ln             net.Listener
}


type Conn struct {
	Server *Server
	rwc    net.Conn
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
		go c.Server()
	}
}

func (srv *Server) newConn(rw net.Conn) (c *Conn, err error) {
	c = new(Conn)
	c.Server = srv
	c.rwc = c
	return c, nil
}

// server 单个连接处理函数
func (c *Conn) Server() {

}
