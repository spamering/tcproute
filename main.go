// TcpRoute2 project main.go
package main

import (
	"time"
	"flag"
	"fmt"
	"log"
	"github.com/gamexg/TcpRoute2/netchan"
	"path/filepath"
	"github.com/BurntSushi/toml"
	"os"
	"io"
	"bytes"
)

const version = "0.5.8"

type ServerConfig struct {
	Addr          string `default:"127.0.0.1:7070"`
	UpStreams     []*ConfigDialClient
	PreHttpPorts  []int // 不使用默认值，好能检测配置文件是否有这个配置项
	PreHttpsPorts []int
	Hosts         []*netchan.DnschanHostsConfigHosts
}


func main() {
	printVer := flag.Bool("version", false, "print version")
	config_path_flag := flag.String("config", "config.toml", "配置文件路径")
	flag.Parse()

	fmt.Println("TcpRoute2 version", version)
	if *printVer {
		return
	}

	config_path, err := filepath.Abs(*config_path_flag)
	if err != nil {
		log.Fatal("配置文件路径错误：", err)
	}
	config_dir := filepath.Dir(config_path)

	// 打开配置文件
	configFile, err := os.Open(config_path)
	if err != nil {
		log.Fatalf("无法打开配置文件(%v)，错误：%v", config_path, err)
	}

	// 解决 Near line 0 (last key parsed ''): Bare keys cannot contain '\ufeff'. 错误。
	buf := make([]byte, 3)
	if _, err := io.ReadFull(configFile, buf); err != nil {
		log.Fatalf("读取配置文件(%v)错误：%v", config_path, err)
	}
	// https://zh.wikipedia.org/wiki/%E4%BD%8D%E5%85%83%E7%B5%84%E9%A0%86%E5%BA%8F%E8%A8%98%E8%99%9F
	if bytes.Equal(buf, []byte{0xEF, 0xBB, 0xBF}) == false {
		configFile.Seek(0, 0)
	}

	// 读取配置
	serverConfig := ServerConfig{}
	_, err = toml.DecodeReader(configFile, &serverConfig)
	if err != nil {
		log.Fatal("解析配置文件错误：", err)
	}

	// 判断 客户端dns解析纠正功能
	if len(serverConfig.PreHttpPorts) == 0 && len(serverConfig.PreHttpsPorts) == 0 {
		log.Printf("未配置是否启用 客户端dns解析纠正功能，默认将在发现浏览器进行了dns本地解析时强制改为为代理服务器进行dns解析。")
		serverConfig.PreHttpPorts = []int{80}
		serverConfig.PreHttpsPorts = []int{443}
	}
	preHttpPorts = serverConfig.PreHttpPorts
	preHttpsPorts = serverConfig.PreHttpsPorts

	// 修正一些参数的默认值
	// multiconfig 第三方库存在 bug ，部分配置默认值无效，先临时修正，之后检查第三方库的代码。
	// 大概是 toml 库的问题。
	for _, v := range serverConfig.Hosts {
		if v.UpdateInterval == "" {
			v.UpdateInterval = "24h"
		}

		if v.Type == "" {
			v.Type = "base"
		}
	}

	for _, v := range serverConfig.UpStreams {
		if v.ProxyUrl == "" {
			v.ProxyUrl="direct://0.0.0.0:0000"
		}
		if v.Name == "" {
			v.Name = v.ProxyUrl
		}
		wbListFunc := func(list    []*ConfigDialClientWBList) error {
			for _, v := range list {
				if v.Path == "" {
					return fmt.Errorf("黑白名单 Path 不能为空")
				}
				if v.Type == "" {
					v.Type = "base"
				}
				if v.UpdateInterval == "" {
					v.UpdateInterval = "24h"
				}
			}
			return nil
		}
		if err := wbListFunc(v.Blacklist); err != nil {
			log.Println("黑名单配置错误：", err)
		}
		if err := wbListFunc(v.Whitelist); err != nil {
			log.Println("白名单配置错误：", err)
		}
	}



	if err := netchan.HostsDns.Config(&netchan.DnschanHostsConfig{BashPath:config_dir,
		Hostss:serverConfig.Hosts,
		CheckInterval:1 * time.Minute,
	}); err != nil {
		log.Print(err)
		return
	}

	// 获得线路列表
	configDialClients := ConfigDialClients{
		UpStreams:serverConfig.UpStreams,
		BasePath:config_dir,
	}

	dialClients, err := NewDialClients(&configDialClients)
	if err != nil {
		log.Print(err)
		return
	}

	// 创建 tcpping 上层代理
	upStream := NewTcppingUpStream(dialClients)




	// 服务器监听
	srv := NewServer(serverConfig.Addr, upStream)

	// TODO: DNS 配置

	// TODO: 各端口需要的安全级别

	if err := srv.ListAndServe(); err != nil {
		log.Print(err)
		return
	}
}

