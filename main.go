// TcpRoute2 project main.go
package main

import (
	"github.com/golang/glog"
	"github.com/koding/multiconfig"
	"time"
	"flag"
	"fmt"
)

const version = "0.1.0"

type ServerConfig struct {
	Addr      string `default:":5050"`
	UpStreams []ServerConfigUpStream
}

type ServerConfigUpStream struct {
	Name       string`default:"direct"`
	ProxyUrl   string`default:"direct://0.0.0.0:0000"`
	DnsResolve bool `default:"false"`
	Credit     int `default:"0"`
	Sleep      int `default:"0"`
	CorrectDelay  int `default:"0"`
}


func main() {
	defer glog.Flush()
	//os.Setenv("GLOG_logtostderr", "1")
	//os.Setenv("GLOG_stderrthreshold", "0")

	printVer :=	flag.Bool("version", false, "print version")
	config_path := flag.String("config", "config.toml", "配置文件路径")
	flag.Parse()

	if *printVer{
		fmt.Println("TcpRoute2 version", version)
		return
	}

	m := multiconfig.NewWithPath(*config_path)

	serverConfig := new(ServerConfig)
	m.MustLoad(serverConfig)

	// 服务器监听
	srv := NewServer(serverConfig.Addr)

	// 创建 tcpping 上层代理
	upStream := NewTcppingUpStream(srv)
	srv.upStream = upStream

	for _, up := range serverConfig.UpStreams {
		if err := upStream.AddUpStream(up.Name, up.ProxyUrl, up.DnsResolve, up.Credit, time.Duration(up.Sleep) * time.Millisecond, time.Duration(up.CorrectDelay) * time.Millisecond); err != nil {
			panic(err)
		}
	}

	// DNS 配置

	// 各端口需要的安全级别

	srv.ListAndServe()
}

