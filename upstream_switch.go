package main
import (
	"time"
	"net"
	"github.com/gamexg/proxyclient"
	"fmt"
)

// 切换器
type switchUpStream struct {
	srv *Server
	pc  proxyclient.ProxyClient
}

func NewSwitchUpStream(srv *Server) (*switchUpStream, error) {
	pc, err := proxyclient.NewProxyClient("http://127.0.0.1:7777")
	if err != nil {
		return nil, fmt.Errorf("无法创建上层代理：%v", err)
	}
	return &switchUpStream{srv, pc }, nil
}

func (su*switchUpStream)DialTimeout(network, address string, timeout time.Duration) (net.Conn, error) {
	return su.pc.DialTimeout(network, address, timeout)
}
