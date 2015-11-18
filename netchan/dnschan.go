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
		defer func(){_=recover()}()
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


	for _, ip := range ips {
		RecordChan <- &DnsRecord{ip.String(), 0}
	}


}
func (s *systemDNS)querySleep() time.Duration {
	return 0
}

