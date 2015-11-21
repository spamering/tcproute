// TcpRoute2 project main.go
package main

import (
	"os"
	"github.com/golang/glog"
//	"flag"
)

func main() {
	//flag.Parse()
	defer glog.Flush()

	os.Setenv("GLOG_logtostderr", "1")
	os.Setenv("GLOG_stderrthreshold", "0")


	srv := NewServer(":7070")

	// tcpping 上层代理
	upStream, err := NewTcppingUpStream(srv)
	if err != nil {
		panic(err)
	}
	srv.upStream = upStream

	srv.ListAndServe()
}

