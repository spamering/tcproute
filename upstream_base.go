package main
import (
	"time"
	"net"
	"github.com/gamexg/proxyclient"
	"fmt"
)

// 切换器
type baseUpStream struct {
	srv *Server
	pc  proxyclient.ProxyClient
}

func NewBaseUpStream(srv *Server) (*baseUpStream, error) {
	pc, err := proxyclient.NewProxyClient("http://127.0.0.1:7777")
	if err != nil {
		return nil, fmt.Errorf("无法创建上层代理：%v", err)
	}
	return &baseUpStream{srv, pc }, nil
}

func (su*baseUpStream)DialTimeout(network, address string, timeout time.Duration) (net.Conn, error) {
	return su.pc.DialTimeout(network, address, timeout)
}
