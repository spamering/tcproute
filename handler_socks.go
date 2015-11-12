package main
import (

	"fmt"
	"net"
)

// socks 服务器
type hSocksServer struct {

}

type hSocksHandle struct {
	server *hSocksServer
}

func NewSocksHandlerNewer() HandlerNewer {
	return &hSocksServer{}
}

// 尝试创建 socks 处理器
func (sev *hSocksServer) New(c net.Conn) (h Handler, reset bool, err error) {
	// 读取前几个字节，判断是否为 socks 代理协议。
	// 是则返回 handler
	reset = true
	b := make([]byte, 1024)

	if n, err := c.Read(b); err != nil {
		err = fmt.Errorf("读取错误：%v", err)
		return
	}else {
		b = b[:n]
	}

	if b[0] == 0x04 {
		err = fmt.Errorf("暂时不支持 socks4 协议。")
		return
	}else if b[0] == 0x05 {
		return &hSocksHandle{sev}, true, nil
	}else {
		err = NoHandleError("协议头不正确。")
		return
	}
}

func (h *hSocksHandle)String() string {
	return "Socks5"
}

// 实际处理 Socks5 协议
func (h*hSocksHandle)Handle() error {
	// 实际处理 socks 协议。
	// 这里需要转发给下一级处理器。

	// 对于 80 、443 端口的连接可以尝试

	panic("未完成")
}
