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

// 基本的错误报告实现
type UpStreamErrorReportingBase struct {
	errConnServer                *ErrConnService
	dailName, DomainAddr, IpAddr string

}

func (er*UpStreamErrorReportingBase) Report(t ErrConnType) {
	er.errConnServer.AddErrLog(er.dailName, er.DomainAddr, er.IpAddr, t)
}

// 错误报告
type UpStreamErrorReporting interface {
	// 使用者认为连接有问题时调用
	Report(t ErrConnType)
}


type UpStreamDial interface {
	// 建立新连接
	// 如果没有实现 UpStreamErrorReporting 可以返回 nil
	DialTimeout(network, address string, timeout time.Duration) (net.Conn, UpStreamErrorReporting, error)


}
