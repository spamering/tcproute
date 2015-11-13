package main
import (
	"time"
	"net"
)

/*

为了实现 http 、https 连接异常自动切换连接IP的功能，必须接受连接并读取请求内容。

大概和标准的 tcp 连接一样即可

先尝试建立连接，对于非 http、https 是真正的尝试建立连接，建立连接后才返回建立连接成功。

对于 http、https 连接是直接返回建立连接成功，然后接收 http 请求内容，根据http 请求的 host 或 https 的 SNI 来获得目标网站的地址。这样可以防止dns欺骗。


*/

type UpStream interface {

}



type UpStreamDial interface {
	// 建立新连接
	DialTimeout(network, address string, timeout time.Duration) (net.Conn, error)
}
