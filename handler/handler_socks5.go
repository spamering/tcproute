package main
import (

)

// socks5 服务器
type HSocks5Server struct {

}

type HSocks5Handle struct {

}

// 创建 socks5 处理器
func (sev *HSocks5Server) New(c Conn) (h Handler,reset bool, err error) {
	// 读取前几个字节，判断是否为 socks 代理协议。
	// 是着返回 handler
	panic("未完成")
}

func (h *HSocks5Handle)String() string {
	return "Socks5"
}

// 实际处理 Socks5 协议
func (h*HSocks5Handle)Handle() error{
	// 实际处理 socks 协议。
	// 这里需要转发给下一级处理器。

	// 对于 80 、443 端口的连接可以尝试

	panic("未完成")
}
