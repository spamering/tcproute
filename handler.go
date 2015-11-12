package main
import (
	"fmt"
	"net"
)


// NoHandle 无法处理的协议类型
// 尝试通过 New 对连接创建 Handler 时，如果协议不匹配无法处理，那么就返回这个错误。
type NoHandleError string

func (e * NoHandleError) Error() string {
	return fmt.Sprintf("协议无法处理：%v", string(e))
}

type Handler interface {
	String() string
	// Handler 是实际处理请求的函数
	// 注意：如果返回那么连接就会被关闭。
	// 注意：默认设置了10分钟后连接超时，如需修改请自行处理。
	Handle()
}

// Newer 是创建处理器接口
// 如果处理器识别到当前连接可以处理，那么就会返回创建的处理器，否则返回 nil
type HandlerNewer interface {
	// New 创建处理器
	// 有可能由于协议不正确会创建失败，这时 h==nil , error 可以返回详细信息
	// 在创建处理器失败时调用方负责回退已经从 stream 读取的数据
	// 在创建处理器成功时根据reset的值确定是否复位预读。true 复位预读的数据，flase不复位预读的数据。
	// 注意：函数内部不允许调用会引起副作用的方法，例如 close、 write 等函数 。
	New(conn net.Conn) (h Handler, rePre bool, err error)
}
