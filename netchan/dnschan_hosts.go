package netchan
import (
	"sync"
	"time"
	"log"
	"fmt"
	"strings"
	"bufio"
	"io"
	"github.com/gamexg/TcpRoute2/domains"
	"github.com/gamexg/TcpRoute2/ufile"
)

/*
hosts 文件
*/

const hostsCheckInterval = 1 * time.Minute //http检测间隔
const hostsCacheSize = 200

type hostsDns struct {
	uf  *ufile.UFile
	ds  *domains.Domains
	rwm sync.RWMutex
}


//配置(对外)
type DnschanHostsConfig struct {
	BashPath      string        // hosts 路径前缀
	CheckInterval time.Duration //http检测间隔
	Hostss        []*DnschanHostsConfigHosts
}
type DnschanHostsConfigHosts struct {
	Path           string `required:"true"` // host 文件路径、本地或远程url(http、https)
	UpdateInterval string `default:"24h"`   // 更新间隔、当使用远程 url 时检查更新间隔
	Credit         int    `default:"0"`     // dns信誉
	Type           string `default:"base"`
}

type ufileUserdata struct {
	name       string // 唯一名称。 目前是通过 type+credit+path 标识的。
	domainType domains.DomainType
	credit     int
}

type domainUserdata struct {
	fileName string // 所属的文件名称（类型+信誉+路径）
	credit   int
	ips      []string
}

func NewHostsDns(c *DnschanHostsConfig) (*hostsDns, error) {

	h := hostsDns{}

	if err := h.Config(c); err != nil {
		return nil, err
	}

	return &h, nil
}

type hostQueryTem struct {
	credit int
	ips    []string
}

func (h*hostsDns)query(domainName string, RecordChan chan *DnsRecord, ExitChan chan int) {
	var domain *domains.Domain

	// 读出结果
	func() {
		h.rwm.RLock()
		defer h.rwm.RUnlock()
		domain = h.ds.Find(domainName)
	}()

	// 输出结果
	for _, userdata := range domain.Userdatas {
		duserdata := userdata.(*domainUserdata)
		for _, ip := range duserdata.ips {
			func() {
				defer func() {_ = recover()}()
				select {
				case <-ExitChan:
					return
				default:
					RecordChan <- &DnsRecord{ip, duserdata.credit}
				}
			}()
		}}

}

// 更新配置
// hosts 本地文件会直接载入
//       不存在时打印警告，但是不出错退出，等新建 hosts 文件时将会自动载入
// hosts http、https 文件，定时更新。
// 每次配置重新载入所有的配置文件，并不是函数返回配置就立刻生效，配置生效需要时间。
func (h*hostsDns)Config(c *DnschanHostsConfig) error {
	newUFile, err := ufile.NewUFile(c.BashPath, hostsCheckInterval)
	if err != nil {
		return err
	}

	for _, hosts := range c.Hostss {
		if strings.TrimSpace(hosts.Type) == "" {
			hosts.Type = "base"
		}
	}

	for _, hosts := range c.Hostss {
		// 解析更新间隔
		if lpath := strings.ToLower(hosts.Path);
		hosts.UpdateInterval == "" &&
		strings.HasPrefix(lpath, "http://") == false&&
		strings.HasPrefix(lpath, "https://") == false {
			// 本地文件不存在更新间隔时自动补一个，其实没有用处。
			hosts.UpdateInterval = "1h"
		}

		updateInterval, err := time.ParseDuration(hosts.UpdateInterval)
		if err != nil {
			newUFile.Close()
			return fmt.Errorf("hosts 配置错误：updateInterval (%v) 格式不是正确的时间格式：%v", hosts.UpdateInterval, err)
		}

		// 类型
		domainType, err := domains.ParseDomainType(hosts.Type)
		if err != nil {
			return fmt.Errorf("未知的 hosts 类型：%v", hosts.Type)
		}

		// userdata
		name := fmt.Sprint(domainType, "-", hosts.Credit, "-", hosts.Path)
		userdata := ufileUserdata{
			name:name,
			domainType:domainType,
			credit:hosts.Credit,
		}

		if err := newUFile.Add(hosts.Path, updateInterval, &userdata); err != nil {
			newUFile.Close()
			return err
		}
	}


	// 配置没有明显错误，使用新配置
	h.rwm.Lock()
	defer h.rwm.Unlock()

	if h.uf != nil {
		h.uf.Close()  //关闭老配置的自动更新
	}

	newDomains := domains.NewDomains(hostsCacheSize)

	h.ds = newDomains
	h.uf = newUFile

	go h.loop(newUFile, newDomains)

	return nil
}


func (h*hostsDns)loop(uf *ufile.UFile, ds*domains.Domains) {
	for file := range uf.ResChan {
		if file.Err != nil {
			log.Printf("载入 hosts文件(%v) 失败，错误：%v", file.Path, file.Err)
			continue
		}

		// 解析文件
		hosts, err := LoadHostsStream(file.Rc)
		if err != nil {
			log.Printf("载入 hosts文件(%v) 失败，错误：%v", file.Path, err)
			continue
		}

		userdata := file.Userdata.(*ufileUserdata)

		func() {
			h.rwm.Lock()
			defer h.rwm.Unlock()

			// 删除旧的记录
			ds.RemoveType(userdata.domainType, func(domain string, domainType domains.DomainType, domainUesrdata domains.UserData) bool {
				lDomainUserdata := domainUesrdata.(*domainUserdata)
				if lDomainUserdata.fileName == userdata.name {
					return true
				}
				return false
			})

			// 添加新记录
			for domainName, ips := range hosts {
				dUserdata := domainUserdata{
					fileName:userdata.name,
					credit:userdata.credit,
					ips:ips,
				}

				ds.Add(domainName, userdata.domainType, &dUserdata)
			}
		}()
		log.Printf("载入 hosts文件成功(%v) ", file.Path)
	}
}



func (h*hostsDns)Close() {
	h.rwm.Lock()
	defer h.rwm.Unlock()

	if h.uf != nil {
		h.uf.Close()
		h.uf = nil
		h.ds = nil
	}
}


// 载入 hosts
// 不处理泛解析及正则解析
// 由于不知道之后由什么处理，并不会进行小写转换。
// 不会检查 ip 格式是不是合法，计划不合法的ip 当作 CNAME 处理。
func LoadHostsStream(f io.Reader) (d map[string][]string, err error) {
	d = make(map[string][]string)

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		rawText := scanner.Text()

		// 跳过注释及空行
		t := strings.TrimSpace(rawText)
		if t == "" || t[0] == '#' {
			continue
		}

		// 替换 制表符 \t 为空白
		t = strings.Replace(t, "\t", " ", -1)

		// 拆分
		ts := strings.SplitN(t, " ", 2)

		// 跳过错误行
		if len(ts) != 2 {
			log.Printf("解析hosts文件，发现错误行，跳过。行内容：%v", rawText)
			continue
		}

		ip := strings.TrimSpace(ts[0])
		domain := strings.TrimSpace(ts[1])

		d[domain] = append(d[domain], ip)
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return
}

