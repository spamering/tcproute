package main
import (
	"io"
)

// Conn 表示连接
type Conn interface {
	reader() io.Reader
	writer() io.Writer
	closer() io.Closer
}

type HContext interface {
}


// NoHandle 无法处理的协议类型
// 尝试通过 New 对连接创建 Handler 时，如果协议不匹配无法处理，那么就返回这个错误。
type NoHandleError int

func (e * NoHandleError) Error() string {
	return "协议无法处理。"
}

type Handler interface {
	// Handler 是实际处理请求的函数
	String() string
	Handle()
}

// Newer 是创建处理器接口
type Newer interface {
	// New 创建处理器
	// 有可能由于协议不正确会创建失败，这时 error != nil
	// 在失败时调用方负责回退已经从 stream 读取的数据
	// reset 表示在调用 Handle 前是否复位 stream
	// 注意：函数内部不允许调用回忆起副作用的方法，例如 close、 write 等函数 。
	New(Conn) (h Handler, reset bool, err error)
}


