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

	srv := Server{}
	srv.Addr = ":7070"

	// 处理器
	h := NewSwitchHandlerNewer(&srv)
	hs := NewSocksHandlerNewer(&srv)
	h.AppendHandlerNewer(hs)
	srv.hNewer = h

	// 上层代理
	upStream, err := NewSwitchUpStream(&srv)
	if err != nil {
		panic(err)
	}
	srv.upStream = upStream

	srv.ListAndServe()
}

