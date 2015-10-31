package main

// socks5 服务器
type HSocks5Server struct {

}

type HSocks5Handle struct {

}

// 创建 socks5 处理器
func (sev *HSocks5Server) New(Conn) (h Handler,reset bool, err error) {
	// 预期的做法是赌气几直接
}

func (h *HSocks5Handle)String() string {
	return "Socks5"
}

// 实际处理 Socks5 协议
func (h*HSocks5Handle)Handle() {

}
