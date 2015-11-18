package main
import (
	"time"
	"net"
	"github.com/gamexg/proxyclient"
	"fmt"
	"github.com/gamexg/TcpRoute2/netchan"
	"github.com/golang/glog"
)

/*

需要建立多个连接，是用最快建立的连接。

要求：
	同时使用多个连接
	限制同时使用的连接总数，不能一味获得了100个 ip 就同时建立 100 个连接
	至少对于本地连接需要执行 dns 解析，尝试获得最快的 ip

实现：
	已经计划通过 chan 返回建立成功的连接，但是总连接数怎么限制？
		是通过阻塞写入 chan 结果信道来实现？
			阻塞结果信道并不可取，只有已经建立连接后才会填充结果队列。
		还是制作一个令牌信道，获得令牌的才可以建立连接？

		不这么麻烦，直接启动多个线程

最终确定：
	并列的代理数量一般不多，这个不管了，直接都尝试连接。
	主要是本地直连部分，由于google的ip数量是以千为单位的，真要一次连接这么多就太坑了，所以会针对这个做个限制。
		同时创建 5 个协程来建立连接。

实现：
	输入参数：
		目标地址(域名)
		输出连接队列
		退出队列


是否设置一个全局每秒新建连接数限制？
	暂时不限制，目前是家庭环境使用，量不是很大，并没必要限制这个。

*/

type DialClient struct {
	dnsResolve bool
	pc         proxyclient.ProxyClient
}

// TcpPing 方式选择最快连接上的。
type tcppingUpStream struct {
	srv *Server
	dc  []DialClient
}

func NewTcppingUpStream(srv *Server) (*tcppingUpStream, error) {
	tUpstream := tcppingUpStream{srv, make([]proxyclient.ProxyClient)}

	// 加入线路
	var addErr error
	addProxyClient := func(proxyUrl string, dnsResolve bool) {
		pc, err := proxyclient.NewProxyClient(proxyUrl)
		if err != nil {
			addErr = fmt.Errorf("无法创建上层代理：%v", err)
			return
		}
		tUpstream.dc = append(tUpstream.dc, DialClient{dnsResolve, pc})
	}

	addProxyClient("direct://0.0.0.0:0000", true)
	addProxyClient("http://127.0.0.1:7777", false)

	if addErr != nil {
		return nil, addErr
	}

	return &tUpstream, nil
}

type dialTimeoutRes struct {
	conn net.Conn
	err  error
}

func (su*tcppingUpStream)DialTimeout(network, address string, timeout time.Duration) (net.Conn, error) {
	// 循环各个代理建立连接

	resChan := make(dialTimeoutRes)
	connChan := make(chan netchan.ConnRes)
	exitChan := make(chan int)

	// 另开一个线程进行连接并整理连接信息
	go func() {

		// 退出时不再进行新连接
		defer func() {
			defer func() { _ = recover() }()
			close(exitChan)
		}()

		// 循环使用各个 upstream 进行连接
		goConnEndChan := make(chan int)
		for _, d := range su.dc {
			d := d
			go func() {
				defer func() {goConnEndChan <- 1}()
				cerr := netchan.ChanDialTimeout(d.pc, connChan, exitChan, d.dnsResolve, network, address, timeout)
				if cerr != nil {
					glog.Warning(cerr)
				}
			}()
		}

		// 所有连接线程都结束时关闭 connChan 信道
		// 终止取结果的线程，防止永久阻塞。
		go func() {
			for i := 0; i < len(su.dc); i++ {
				<-goConnEndChan
			}
			close(connChan)
		}()

		// 取结果
		// 将最快建立的结果返回给 resChan 好返回主函数。
		// 在无法建立连接时将返回err
		go func() {
			// 取结果
			for conn := range connChan {
				resChan <- dialTimeoutRes{conn.Conn, nil}
				//TODO: 这里可以保存最快的ip，下次就不需要再尝试各个连接了。
				return
			}
			resChan <- dialTimeoutRes{nil, fmt.Errorf("所有线路建立连接失败。")}
		}()
	}()

	return <-resChan
}
