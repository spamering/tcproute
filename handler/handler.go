package handler
import "github.com/gamexg/TcpRoute2/server"

type Handler interface {
	// Handler 是实际处理请求的函数
	Handler()
}


// Newer 是创建处理器接口
type Newer interface {
	// New 创建处理器
	// 有可能由于协议不正确会创建失败，这时 error != nil
	// 在失败时调用方负责回退已经从 stream 读取的数据
	// 注意：函数内部不允许调用回忆起副作用的方法，例如 close、 write 等函数 。
	New(server.Conn) (Handler,error)
}

