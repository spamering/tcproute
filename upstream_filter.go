package main
import (
	"time"
	"net"
	"fmt"
	"strings"
	"strconv"
	vhost "github.com/peacekeeper/golang-github-inconshreveable-go-vhost-dev"
	"github.com/gamexg/proxyclient"
)


// 过滤器
// 目前的功能是当前端应用执行了本地 DNS 查询时强制改为代理执行 DNS 查询。
// 主要用于配合 Proxifier 和 redsocks 。
// 实现方式为：当目标地址是 IP ，端口是 80 、443 时，会读取协议头获得目标域名。
// 注意：https 客户端需要 SNI 支持才能为 https 协议起作用（Windoes XP 下全部 IE 版本不支持)。
type filterUpStream struct {
	upStream *UpStreamDial
}

// 坑死，这样不就需要自己实现所有的 TCPConn 方法，很多麻烦。
// 考虑错误，应该放到 handler 去实现。
type filterTcpConn struct {
	proxyclient.TCPConn
	upStream *filterUpStream
	address  string
	protocol int
}


func NewFilterUpStream(upStream *UpStreamDial, httpProts, HttpsProts[]int) (*baseUpStream, error) {
	if upStream == nil {
		return nil, fmt.Errorf("upStream 不可为空。")
	}

	return filterUpStream{upStream:upStream, httpProts:httpProts, HttpsProts:HttpsProts}, nil
}

func (f*filterUpStream)DialTimeout(network, address string, timeout time.Duration) (net.Conn, UpStreamErrorReporting, error) {
	if strings.HasPrefix(network, "tcp") == false {
		// 非 tcp 协议不处理
		goto forward
	}

	host, port, err := net.SplitHostPort(address)
	if err != nil {
		// 地址异常不处理
		goto forward
	}

	ip := net.ParseIP(host)
	if ip == nil {
		// 目标地址非 ip 不处理
		goto forward
	}

	portInt, err := strconv.Atoi(port)
	if err != nil {
		// 端口异常不处理
		goto forward
	}

	if in(portInt, f.httpProts) {
		// 匹配 http ，处理

	}



	forward:
	return f.DialTimeout(network, address, timeout)
}
func in(v int, l  []int) bool {
	for i := range (l) {
		if v == i {
			return true
		}
	}
	return false
}