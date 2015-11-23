package netchan
import (
	"time"
	"net"
	"fmt"
	"sync/atomic"
)


const LocalConnGoCount = 30 // 本地连接时使用的线程数（只对dns解析结果生效）

type ConnRes struct {

	Dial       DialTimeouter
	Conn       net.Conn
	Ping       time.Duration // 连接耗时
	DomainAddr string
	IpAddr     string        //当使用DNS解析时，保存着 ip:端口 格式的地址
	UserData   interface{}
}

type DialTimeouter interface {
	DialTimeout(network, address string, timeout time.Duration) (net.Conn, error)
}

/*
将标准的 Dial 接口转换成 Chan 返回。

可以通过选项指定是否本地dns解析，使用本地解析时会同时使用获得的多个ip连接，同样通过 chan 返回所有建立的连接。


*/
func ChanDialTimeout(dial DialTimeouter, connChan chan ConnRes, exitChan chan int, dnsResolve bool, userData interface{}, network, address string, timeout time.Duration) (rerr error) {
	myExitChan := make(chan int)
	defer close(myExitChan)

	select {
	case <-exitChan:
		return nil
	default:

	// 检查是否使用的ip地址。
		host, prot, err := net.SplitHostPort(address)
		if err != nil {
			return fmt.Errorf("地址错误：%v", err)
		}
		ip := net.ParseIP(host)

		if dnsResolve == false || ip != nil {
			n := time.Now()
			c, err := dial.DialTimeout(network, address, timeout)
			if err != nil {
				return err
			}else {
				func() {
					defer func() {_ = recover()}()
					connChan <- ConnRes{dial, c, time.Now().Sub(n), address, address, userData}
				}()
				return nil
			}
		}else {
			// 本地执行 DNS 解析
			dnsRes := NewDnsQuery(host)

			// 退出时及被要求终止时停止dns解析
			go func() {
				defer func() {_ = recover()}()
				select {
				case <-myExitChan:
					dnsRes.Stop()
				case <-exitChan:
					dnsRes.Stop()
				}
			}()

			// 启动多个连接线程连接
			goEndChan := make(chan int)
			var okCount uint32 = 0
			for i := 0; i < LocalConnGoCount; i++ {
				go func() {
					defer func() { goEndChan <- 0 }()
					for r := range dnsRes.RecordChan {
						select {
						case <-exitChan:
							return
						default:
							ipAddr := net.JoinHostPort(r.Ip, prot)
							n := time.Now()
							c, err := dial.DialTimeout("tcp", ipAddr, timeout)
							if err != nil {
								rerr = err
								continue
							}
							atomic.AddUint32(&okCount, 1)

							func() {// 使用匿名函数捕获异常，防止 connChan 关闭时崩溃
								defer func() {_ = recover()}()
								connChan <- ConnRes{dial, c, time.Now().Sub(n), address, ipAddr, userData}
							}()
						}
					}
				}()
			}

			// 等待所有线程运行完毕
			for i := 0; i < LocalConnGoCount; i++ {
				<-goEndChan
			}

			if okCount > 0 {
				return nil
			}
			return
		}
	}
}
