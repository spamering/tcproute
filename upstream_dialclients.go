package main
import (
	"github.com/gamexg/proxyclient"
	"time"
	"github.com/gamexg/TcpRoute2/domains"
	"sync"
	"github.com/gamexg/TcpRoute2/ufile"
	"fmt"
	"io"
	"bufio"
	"strings"
	"net"
	"log"
)


/*

多线路管理

支持黑白名单功能


*/
type DialClient struct {
	name         string
	dnsResolve   bool
	pc           proxyclient.ProxyClient
	dialCredit   int              // 线路可靠(信誉)程度
	sleep        time.Duration    // 线路在使用前等待的时间
	correctDelay time.Duration    // 对 tcping 修正
	whiteList    *domains.Domains //白名单
	blackList    *domains.Domains //黑名单
}

type DialClients struct {
	rwm     sync.RWMutex
	clients []*DialClient
	uf      *ufile.UFile
}

// 对外配置
type ConfigDialClients struct {
	BasePath  string
	UpStreams []*ConfigDialClient
}

type ConfigDialClient struct {
	Name         string`default:""`
	ProxyUrl     string`default:"direct://0.0.0.0:0000"`
	DnsResolve   bool `default:"false"`    //本线路是否使用本地dns解析，建议对直连、socks4线路进行本地dns解析。
	Credit       int `default:"0"`
	Sleep        int `default:"0"`
	CorrectDelay int `default:"0"`
	Whitelist    []*ConfigDialClientWBList //白名单
	Blacklist    []*ConfigDialClientWBList //黑名单
}

type ConfigDialClientWBList struct {
	Path           string `required:"true"` // 文件路径、本地或远程url(http、https)
	UpdateInterval string `default:"24h"`   // 更新间隔、当使用远程 url 时检查更新间隔
	Type           string `default:"base"`
}


func NewDialClients(config*ConfigDialClients) (*DialClients, error) {
	clients := DialClients{}
	if err := clients.Config(config); err != nil {
		return nil, err
	}

	return &clients, nil
}

type ufUserdata struct {
	name       string
	domainType domains.DomainType
	domains    *domains.Domains
}

func (d*DialClients)Config(config*ConfigDialClients) (rerr error) {

	newClients := make([]*DialClient, 0, len(config.UpStreams))

	newUf, err := ufile.NewUFile(config.BasePath, 1 * time.Minute)
	if err != nil {
		return err
	}
	// 出错退出时关闭 UFile
	defer func() {
		if rerr != nil {
			newUf.Close()
		}
	}()

	// 循环处理每个线路
	for _, c := range config.UpStreams {

		pc, err := proxyclient.NewProxyClient(c.ProxyUrl)
		if err != nil {
			rerr = fmt.Errorf("无法创建上级代理：%v", err)
			return
		}

		client := DialClient{
			name:c.Name,
			dnsResolve:c.DnsResolve,
			pc:pc,
			dialCredit:c.Credit,
			sleep:time.Duration(c.Sleep) * time.Millisecond,
			correctDelay:time.Duration(c.CorrectDelay) * time.Millisecond,
			whiteList:domains.NewDomains(100),
			blackList:domains.NewDomains(100),
		}

		// 循环处理每个黑白名单文件
		// 添加到 ufile 里面，实际重新加载在 ufile 结果信道处处理。
		flist := func(clientList []*ConfigDialClientWBList, wbList *domains.Domains) error {
			for _, f := range clientList {
				UpdateInterval, err := time.ParseDuration(f.UpdateInterval)
				if err != nil {
					return fmt.Errorf("UpdateInterval 时间无法解析，当前内容：%v，错误：%v", f.UpdateInterval, err)
				}


				domainType, err := domains.ParseDomainType(f.Type)
				if err != nil {
					return err
				}

				userdara := ufUserdata{
					name:fmt.Sprintf("%v-%v", f.Type, f.Path),
					domainType:domainType,
					domains:wbList,
				}

				if err := newUf.Add(f.Path, UpdateInterval, &userdara); err != nil {
					return err
				}
			}
			return nil
		}

		if err := flist(c.Whitelist, client.whiteList); err != nil {
			rerr = err
			return
		}
		if err := flist(c.Blacklist, client.blackList); err != nil {
			rerr = err
			return
		}

		newClients = append(newClients, &client)
	}


	d.rwm.Lock()
	defer d.rwm.Unlock()
	d.clients = newClients

	if d.uf != nil {
		d.uf.Close()
	}
	d.uf = newUf

	go d.loop(newUf)

	return nil
}

func (d*DialClients)loop(uf *ufile.UFile) {
	for r := range uf.ResChan {
		if r.Err != nil || r.Rc == nil {
			log.Printf("载入 hosts文件(%v) 失败，错误：%v", r.Path, r.Err)
			continue
		}
		defer r.Rc.Close()

		userdata := r.Userdata.(*ufUserdata)

		// 读取新域名列表
		domainList := loadDomains(r.Rc)

		//删除旧的，domains 内置锁，多线程安全
		userdata.domains.RemoveType(userdata.domainType,
			func(domain string, domainType domains.DomainType, domainUserdata domains.UserData) bool {
				fname := domainUserdata.(string)
				if fname == userdata.name {
					return true
				}
				return false
			})

		// 添加新的，domains 内置锁，多线程安全
		for _, domain := range domainList {
			userdata.domains.Add(domain, userdata.domainType, userdata.name)
		}
		log.Printf("载入域名文件(%v) 成功。", r.Path)
	}
}


func loadDomains(rc io.ReadCloser) ([]string) {
	res := make([]string, 0)

	scanner := bufio.NewScanner(rc)
	for scanner.Scan() {
		rawText := scanner.Text()

		// 跳过注释及空行
		t := strings.TrimSpace(rawText)
		if len(t) == 0 || t[0] == '#' {
			continue
		}

		res = append(res, t)
	}
	return res
}

// 获得指定地址的可用线路列表
// bool 返回 True 时表示使用了黑白名单。
func (d*DialClients)Get(addr string) ([]*DialClient, bool) {
	host, _, err := net.SplitHostPort(addr)
	if err != nil {
		host = addr
	}

	d.rwm.RLock()
	defer d.rwm.RUnlock()

	res := make([]*DialClient, 0, len(d.clients))
	white := false// 是否存在白名单

	for _, client := range d.clients {
		if len(client.whiteList.Find(host).Userdatas) != 0 {
			// 白名单
			if white == false {
				white = true
				res = make([]*DialClient, 0)
			}
			res = append(res, client)
		}else {
			// 不存在白名单，并且不在黑名单里面
			if white == false && len(client.blackList.Find(host).Userdatas) == 0 {
				res = append(res, client)
			}
		}
	}

	if len(res) == len(d.clients) {
		return res, false
	}else {
		return res, true
	}
}