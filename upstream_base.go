package main
import (
	"time"
	"net"
	"github.com/gamexg/proxyclient"
	"fmt"
)

// 切换器
type baseUpStream struct {
	srv      *Server
	pc       proxyclient.ProxyClient
	dialName string
}

func NewBaseUpStream(srv *Server) (*baseUpStream, error) {
	url := "http://127.0.0.1:7777"
	pc, err := proxyclient.NewProxyClient(url)
	if err != nil {
		return nil, fmt.Errorf("无法创建上层代理：%v", err)
	}
	return &baseUpStream{srv, pc, url }, nil
}

func (su*baseUpStream)DialTimeout(network, address string, timeout time.Duration) (net.Conn, UpStreamErrorReporting, error) {
	c, err := su.pc.DialTimeout(network, address, timeout)
	// 其实这里提供这个错误功能并没有意义，单线路即使有错误还是只能使用这一个线路。
	// 不过这个本来就只是演示，所以这个错误报告也是个演示了...
	er := UpStreamErrorReportingBase{su.srv.errConn, su.dialName, address, address}
	return c, &er, err
}
