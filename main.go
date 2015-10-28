// TcpRoute2 project main.go
package main

import (
	"github.com/golang/glog"
	"github.com/gamexg/TcpRoute2/server"
)

func main() {
	defer glog.Flush()

	var srv Server
	srv.Addr=":7070"
	srv.ListAndServe()
}
