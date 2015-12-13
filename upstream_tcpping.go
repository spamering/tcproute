package main
import (
	"time"
	"net"
	"github.com/gamexg/proxyclient"
	"fmt"
	"github.com/gamexg/TcpRoute2/netchan"
	"github.com/golang/glog"
	"strconv"
	"sync"
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
	name       string
	dnsResolve bool
	pc         proxyclient.ProxyClient
	dialCredit int // 线路可靠(信誉)程度
}

// TcpPing 方式选择最快连接上的。
type tcppingUpStream struct {
	srv       *Server
	dc        []DialClient
	connCache *upStreamConnCache
}

type chanDialTimeoutUserData struct {
	dialName   string
	domainAddr string // connChan 有 domainAddr 还增加这个字段的原因是使用缓存时，
}


func NewTcppingUpStream(srv *Server) (*tcppingUpStream, error) {
	tUpstream := tcppingUpStream{srv, make([]DialClient, 0, 5), NewUpStreamConnCache(srv)}

	// 加入线路
	var addErr error
	addProxyClient := func(proxyUrl string, dnsResolve bool) {
		if addErr != nil {
			return
		}

		pc, err := proxyclient.NewProxyClient(proxyUrl)
		if err != nil {
			addErr = fmt.Errorf("无法创建上级代理：%v", err)
			return
		}

		dialCredit := 0
		creditQuery, ok := pc.GetProxyAddrQuery()["credit"]
		if ok {
			if len(creditQuery) > 1 {
				addErr = fmt.Errorf("代理 credit 重复设置，代理url:%v", proxyUrl)
				return
			}
			dialCreditTem, err := strconv.Atoi(creditQuery[0])
			if err == nil {
				dialCredit = dialCreditTem
			}
		}

		tUpstream.dc = append(tUpstream.dc, DialClient{proxyUrl, dnsResolve, pc, dialCredit})
	}

	addProxyClient("direct://0.0.0.0:0000", true)
	addProxyClient("http://127.0.0.1:7777?credit=-100", false)

	if addErr != nil {
		return nil, addErr
	}

	return &tUpstream, nil
}

type dialTimeoutRes struct {
	conn         net.Conn
	errReporting UpStreamErrorReporting
	err          error
}

func (su*tcppingUpStream)DialTimeout(network, address string, timeout time.Duration) (net.Conn, UpStreamErrorReporting, error) {




	// 尝试使用缓存中的连接
	item, err := su.connCache.GetOptimal(address)
	if err == nil {

		ctimeout := item.TcpPing
		if ctimeout < 100 * time.Millisecond {
			ctimeout += 20 * time.Millisecond
		} else if ctimeout < 500 * time.Millisecond {
			ctimeout += 50 * time.Millisecond
		}else {
			ctimeout += 100 * time.Millisecond
		}

		// 考虑了下，还是使用原始的连接方式，而没有使用 dialChan
		n := time.Now()
		c, err := item.dial.DialTimeout(network, item.IpAddr, ctimeout)
		if err == nil {
			fmt.Printf("缓存命中：%v 代理：%v IP：%v \r\n", address, item.dialName, item.IpAddr)
			go su.connCache.Updata(address, item.IpAddr, time.Now().Sub(n), item.dial, item.dialName)
			ErrorReporting := &UpStreamErrorReportingBase{su.srv.errConn, item.dialName, item.DomainAddr, item.IpAddr}
			return c, ErrorReporting, nil
		} else {
			fmt.Printf("缓存连接失败：%v 代理：%v IP：%v err:%v \r\n", address, item.dialName, item.IpAddr, err)
		}
	}

	// 缓存未命中时同时使用多个线路尝试连接。
	su.connCache.Del(address)

	resChan := make(chan dialTimeoutRes, 1)
	connChan := make(chan netchan.ConnRes, 10)
	exitChan := make(chan int)
	// 选定连接退出函数时不再尝试新的连接
	defer func() {
		defer func() { _ = recover() }()
		close(exitChan)
	}()

	// 另开一个线程进行连接并整理连接信息
	go func() {
		// 循环使用各个 upstream 进行连接
		sw := sync.WaitGroup{}
		sw.Add(len(su.dc))
		for _, d := range su.dc {
			d := d
			go func() {
				defer func() {sw.Done()}()
				userData := chanDialTimeoutUserData{d.name, address}
				cerr := netchan.ChanDialTimeout(d.pc, d.dialCredit, connChan, exitChan, d.dnsResolve, &userData, nil, network, address, timeout)
				if cerr != nil {
					glog.Info(fmt.Sprintf("线路 %v 连接 %v 失败，错误：%v", d.name, address, cerr))
				}
			}()
		}

		// 所有连接线程都结束时关闭 connChan 信道
		// 终止取结果的线程，防止永久阻塞。
		go func() {
			sw.Wait()
			close(connChan)
		}()

		// 取结果
		// 将最快建立的结果返回给 resChan 好返回主函数。
		// 在无法建立连接时将返回err
		go func() {
			ok := false // 是否已经找到最快的稳定连接
			var oconn net.Conn // 保存最快的结果，如果全部的连接都有问题时间使用这个连接
			var ErrorReporting *UpStreamErrorReportingBase //错误报告（实际使用 oconn 连接如果发现问题通过本变量报告错误）

			defer func() {
				// 结束时函数关闭之前保存的最快的连接。
				if oconn != nil {
					oconn.Close()
				}
			}()


			for conn := range connChan {
				// 保存最快的连接（可能并不稳定） 用于应付找不到稳定连接的情况
				conn := conn
				userData := conn.UserData.(*chanDialTimeoutUserData)

				// 上报这个连接建立的速度
				go su.connCache.Updata(userData.domainAddr, conn.IpAddr, conn.Ping, conn.Dial, userData.dialName)

				// 返回最快并稳定的连接
				if ok == false && su.srv.errConn.Check(userData.dialName, userData.domainAddr, conn.IpAddr) == true {
					// 找到了最快的稳定连接
					fmt.Printf("为 %v 找到了最快的稳定连接 %v ，线路：%v.\r\n", userData.domainAddr, conn.IpAddr, userData.dialName)
					ok = true
					ErrorReporting = &UpStreamErrorReportingBase{su.srv.errConn, userData.dialName, userData.domainAddr, conn.IpAddr}
					resChan <- dialTimeoutRes{conn.Conn, ErrorReporting, nil}
				} else {
					// 已经有最快稳定链接 或者 本连接不稳定。

					// 未安全返回时保存最快的一个连接
					if oconn == nil && ok == false {
						// 如果未找到可靠连接并且本连接时最快建立的连接 就先保存下本连接等待备用。
						oconn = conn.Conn
						ErrorReporting = &UpStreamErrorReportingBase{su.srv.errConn, userData.dialName, userData.domainAddr, conn.IpAddr}
					}else {
						conn.Conn.Close()
					}
				}
			}

			// 最后如果连接未建立，并且有最快建立的连接，那么即使他不稳定也是用这个。
			if ok == false && oconn != nil {
				fmt.Printf("为 %v 找到了最快但不稳定连接 %v ，线路：%v.\r\n", ErrorReporting.DomainAddr, ErrorReporting.IpAddr, ErrorReporting.DailName)
				resChan <- dialTimeoutRes{oconn, ErrorReporting, nil}
				oconn = nil
				return
			}

			resChan <- dialTimeoutRes{nil, nil, fmt.Errorf("所有线路建立连接失败。")}
		}()
	}()

	res := <-resChan
	return res.conn, res.errReporting, res.err
}

