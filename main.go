// TcpRoute2 project main.go
package main

import (
	"os"
	"github.com/golang/glog"
)

func main() {
	defer glog.Flush()

	os.Setenv("GLOG_logtostderr","1")
	os.Setenv("GLOG_stderrthreshold","0")

	var srv Server
	srv.Addr=":7070"
	srv.ListAndServe()
}

