// TcpRoute2 project main.go
package main

import (
	"github.com/koding/multiconfig"
	"time"
	"flag"
	"fmt"
)

const version = "0.2.0"

type ServerConfig struct {
	Addr      string `default:":5050"`
	UpStreams []ServerConfigUpStream
	Config    string `default:""`
}

type ServerConfigUpStream struct {
	Name         string`default:""`
	ProxyUrl     string`default:"direct://0.0.0.0:0000"`
	DnsResolve   bool `default:"false"`
	Credit       int `default:"0"`
	Sleep        int `default:"0"`
	CorrectDelay int `default:"0"`
}


func main() {
	printVer := flag.Bool("version", false, "print version")
	config_path := flag.String("config", "config.toml", "配置文件路径")
	flag.String("addr", ":5050", "绑定地址")
	flag.Parse()

	if *printVer {
		fmt.Println("TcpRoute2 version", version)
		return
	}

	m := multiconfig.NewWithPath(*config_path)

	serverConfig := new(ServerConfig)
	m.MustLoad(serverConfig)

	// 创建 tcpping 上层代理
	upStream := NewTcppingUpStream()

	for _, up := range serverConfig.UpStreams {
		if up.Name == "" {
			up.Name = up.ProxyUrl
		}

		if err := upStream.AddUpStream(up.Name, up.ProxyUrl, up.DnsResolve, up.Credit, time.Duration(up.Sleep) * time.Millisecond, time.Duration(up.CorrectDelay) * time.Millisecond); err != nil {
			panic(err)
		}
	}

	// 服务器监听
	srv := NewServer(serverConfig.Addr, upStream)

	// TODO: DNS 配置

	// TODO: 各端口需要的安全级别

	srv.ListAndServe()
}

