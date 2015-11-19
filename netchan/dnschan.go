package netchan
import (
	"time"
	"net"
	"sync"
	"fmt"
	"github.com/golang/glog"
)

var queries []queryer
var blackIP = make(map[string]bool)
var rwm = sync.RWMutex{}

// 表示dns 查询
type DnsQuery struct {
						// 返回dns查询结果
	RecordChan chan *DnsRecord
						// queries    []queryer
	exitChan   chan int
	Domain     string
	sleepChan  chan int // 延迟信道
}

// 表示 DNS 记录
type DnsRecord struct {
	Ip     string //地址
	Credit int    //信誉 0 是本机标准的DNS解析结果 越高越可信
}

// 各种类型的查询实现
type queryer interface {
	// 执行dns查询(阻塞)
	// 查询结果会通过 RecordChan 返回，注意可能被阻塞。
	// 在需要提前结束时 ExitChan 会被关闭
	// 注意：实现者需要处理 RecordChan 被关闭的情况。
	query(domain string, RecordChan chan *DnsRecord, ExitChan chan int)

	// 执行前的默认等待时间
	querySleep() time.Duration
}

func init() {
	queries = make([]queryer, 0, 10)

	sysDns := systemDNS("")
	queries = append(queries, &sysDns)

	go searchBlackIP()
}

func searchBlackIP() {
	// 每1小时执行一次
	t := time.NewTicker(1 * time.Hour)

	run := func() {
		bIP := make(map[string]bool)

		for i := 0; i < 3; i++ {
			recordChan := make(chan *DnsRecord)
			exitChan := make(chan int, 10)

			go func() {
				for _, q := range queries {
					domain := fmt.Sprint(time.Now().Unix(), "dshsdjhsdsgsevstyhndrdrntrtvsvstbruiuok095g.com")
					q.query(domain, recordChan, exitChan)
				}
				close(recordChan)
			}()

			for r := range recordChan {
				bIP[r.Ip] = true
			}
		}

		func() {
			rwm.Lock()
			defer rwm.Unlock()
			blackIP = bIP
		}()
		glog.Info("发现异常IP：", blackIP)
	}

	run()

	for _ = range t.C {
		run()
	}
}

func (dq*DnsQuery) query() {
	if len(queries) == 0 {
		return
	}
	queries[0].query(dq.Domain, dq.RecordChan, dq.exitChan)
	for _, q := range queries[1:] {
		time.AfterFunc(q.querySleep(), func() {dq.sleepChan <- 1})
		select {
		case <-dq.sleepChan:
			q.query(dq.Domain, dq.RecordChan, dq.exitChan)
		case <-dq.exitChan:
			return
		}
	}
	dq.Stop()
}

// 返回 DNS 记录信道及取消信道
func NewDnsQuery(domain string) *DnsQuery {
	q := DnsQuery{make(chan *DnsRecord), make(chan int), domain, make(chan int, 10)}
	go q.query()
	return &q
}

// 跳过一次延迟
func (dq*DnsQuery) SkipSleep() {
	go func() {
		defer func() {_ = recover()}()
		dq.sleepChan <- 1
		time.AfterFunc(100 * time.Millisecond, func() {
			select {
			case <-dq.sleepChan:
			case <-dq.exitChan:
			}
		})
	}()
}

func (dq*DnsQuery) Stop() {
	defer func() {
		_ = recover()
	}()
	close(dq.exitChan)
	close(dq.RecordChan)
	for _ = range dq.sleepChan {
	}
}

type systemDNS string
func (s *systemDNS)query(domain string, RecordChan chan *DnsRecord, ExitChan chan int) {
	defer func() {
		_ = recover()
	}()

	select {
	case <-ExitChan:
		return
	default:
	}

	ips, err := net.LookupIP(domain)
	if err != nil {
		return
	}
	if len(ips) <= 0 {
		return
	}

	// 转换ip 为字符串格式，同时检查有没有 屏蔽IP
	ipsString := make([]string, 0, len(ips))
	func() {
		rwm.RLock()
		defer rwm.RUnlock()

		for _, ip := range ips {
			ipString := ip.String()

			// 如果查询包含异常IP，清空本次查询的结果
			// 系统dns解析可以全部抛弃，但是全球web分布式dns解析就不能这么干了
			// 因为可能只是某个地区有问题，不能把所有地区一竿子全部打死。
			if blackIP[ipString] == true {
				glog.Warning(fmt.Sprintf("[systemDNS]解析 %v 时发现异常IP %v ，放弃本次 DNS 解析结果。", domain, ipString))
				ipsString = make([]string, 0)
				return
			}

			ipsString = append(ipsString, ipString)
		}
	}()

	for _, ipString := range ipsString {
		RecordChan <- &DnsRecord{ipString, 0}
	}
}

func (s *systemDNS)querySleep() time.Duration {
	return 0
}

