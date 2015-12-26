// TcpRoute2 project main.go
package main

import (
	"github.com/golang/glog"
	"github.com/koding/multiconfig"
	"time"
)

type ServerConfig struct {
	Addr      string `default:":5050"`
	UpStreams []ServerConfigUpStream
}

type ServerConfigUpStream struct {
	Name       string`default:"direct"`
	ProxyUrl   string`default:"direct://0.0.0.0:0000"`
	DnsResolve bool `default:"false"`
	Credit     int `default:"0"`
	Delay      int `default:"0"`
}


func main() {
	//flag.Parse()
	defer glog.Flush()

	//os.Setenv("GLOG_logtostderr", "1")
	//os.Setenv("GLOG_stderrthreshold", "0")

	m := multiconfig.NewWithPath("config.toml")
	serverConfig := new(ServerConfig)
	m.MustLoad(serverConfig)

	// 服务器监听
	srv := NewServer(serverConfig.Addr)

	// 创建 tcpping 上层代理
	upStream := NewTcppingUpStream(srv)
	srv.upStream = upStream

	for _, up := range serverConfig.UpStreams {
		if err := upStream.AddUpStream(up.Name, up.ProxyUrl, up.DnsResolve, up.Credit, time.Duration(up.Delay) * time.Millisecond); err != nil {
			panic(err)
		}
	}

	// DNS 配置

	// 各端口需要的安全级别

	srv.ListAndServe()
}

