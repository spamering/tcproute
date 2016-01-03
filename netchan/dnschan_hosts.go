package netchan
import (
	"sync"
	"time"
	"github.com/go-fsnotify/fsnotify"
	"log"
	"path/filepath"
	"fmt"
	"strings"
	"os"
	"bufio"
	"io"
	"net/http"
)

/*
hosts 文件
*/

const hostsCheckInterval = 1 * time.Minute //http检测间隔

type hostsDns struct {
	hostss        []*dnschanHosts
	rwm           sync.RWMutex
	watcher       *fsnotify.Watcher //监听本地文件修改。实际监听的是hosts所在的目录
									// configCond sync.Cond
	exited        bool              // 是否已退出
	checkInterval time.Duration     //http检测间隔
}

type dnschanHosts struct {
	Path           string              //host 文件路径(绝对路径)、本地或远程url(http、https)
	Local          bool                //本地？远程
	updateInterval time.Duration       // 更新间隔
	Credit         int                 // dns信誉
	data           map[string][]string // 域名数据 先不支持泛域名匹配、正则域名匹配
	utime          time.Time           // 下次更新日期
}

//配置(对外)
type DnschanHostsConfig struct {
	BashPath      string        // hosts 路径前缀
	CheckInterval time.Duration //http检测间隔
	Hostss        []*DnschanHostsConfigHosts
}
type DnschanHostsConfigHosts struct {
	Path           string              //host 文件路径、本地或远程url(http、https)
	UpdateInterval string `default:""` // 更新间隔、当使用远程 url 时检查更新间隔
	Credit         int `default:"0"`   // dns信誉
}

func NewHostsDns(c *DnschanHostsConfig) (*hostsDns, error) {

	h := hostsDns{checkInterval:c.CheckInterval}

	if err := h.Config(c); err != nil {
		return nil, err
	}

	go h.loop()

	return &h, nil
}

type hostQueryTem struct {
	credit int
	ips    []string
}

func (h*hostsDns)query(domain string, RecordChan chan *DnsRecord, ExitChan chan int) {
	ipss := make([]hostQueryTem, 0)

	// 读出结果
	func() {
		h.rwm.RLock()
		defer h.rwm.RUnlock()
		for _, hosts := range h.hostss {
			v, ok := hosts.data[domain]
			if ok {
				// 这里未创建副本的原因是：
				// 主要是每次更新 hosts 都是重新创建，所以不用担心多线程的问题
				ipss = append(ipss, hostQueryTem{hosts.Credit, v})
			}
		}
	}()

	// 输出结果
	for _, r := range ipss {
		for _, ip := range r.ips {
			func() {
				defer func() {_ = recover()}()
				select {
				case <-ExitChan:
					return
				default:
					RecordChan <- &DnsRecord{ip, r.credit}
				}
			}()
		}
	}
}

// 更新配置
// hosts 本地文件会直接载入
//       不存在时打印警告，但是不出错退出，等新建 hosts 文件时将会自动载入
// hosts http、https 文件标记空数据，utime为 0，扫描时更新。
func (h*hostsDns)Config(c *DnschanHostsConfig) error {
	c.BashPath = strings.TrimSpace(c.BashPath)

	hostss := make([]*dnschanHosts, len(c.Hostss))

	hostsDirs := make(map[string]bool) // 保存 hosts 文件夹，监听hosts文件更改并重新载入时使用

	for i, hosts := range c.Hostss {
		// 解析更新间隔
		updateInterval, err := time.ParseDuration(hosts.UpdateInterval)
		if err != nil {
			return fmt.Errorf("hosts 配置错误：updateInterval %v 格式不是正确的时间格式：%v", hosts.UpdateInterval, err)
		}

		hosts.Path = strings.TrimSpace(hosts.Path)
		// 检查路径
		if hosts.Path == "" {
			return fmt.Errorf("hosts 配置错误：Path 不能为空。")
		}

		var data map[string][]string
		local := true

		if strings.HasPrefix(hosts.Path, "http://") || strings.HasPrefix(hosts.Path, "https://") {
			data = make(map[string][]string)
			local = false
		} else {
			// 本地文件

			hosts.Path, err = filepath.Abs(PathJoin(c.BashPath, hosts.Path))
			if err != nil {
				return fmt.Errorf("hosts 配置错误：获得绝对路径失败,当前路径：%v", PathJoin(c.BashPath, hosts.Path))
			}

			// 获得 hosts 文件所在目录
			dir := filepath.Dir(hosts.Path)
			if err := os.MkdirAll(dir, 0711); err != nil {
				return fmt.Errorf("hosts 配置错误：%v hosts 文件所在目录 %v 不存在。", hosts.Path, dir)
			}

			hostsDirs[dir] = true

			// 载入 hosts文件
			data, err = LoadHostsFile(hosts.Path)
			if err != nil {
				if os.IsNotExist(err) {
					log.Printf("hosts 配置错误：%v hosts 文件不存在，将在 hosts 文件创建时载入 hosts 文件。", hosts.Path)
				}else {
					log.Printf("载入hosts文件(%s)失败，错误：%v", hosts.Path, err)
				}
			}
		}
		// 注意： http hsots 时 data 为空。
		// utime 都为空置 0， 扫描http 更新时将会更新 data。
		h := dnschanHosts{Path:hosts.Path, Local:local, updateInterval:updateInterval, Credit:hosts.Credit, data:data}
		hostss[i] = &h
	}

	// 监听 hosts 文件
	nWatcher, err := fsnotify.NewWatcher()
	if err != nil {
		return fmt.Errorf("hosts 配置错误：监听hosts文件变更失败：", err)
	}
	for dir, _ := range hostsDirs {
		if err := nWatcher.Add(dir); err != nil {
			return fmt.Errorf("hosts 配置错误：监听 hosts 文件夹修改失败。文件夹:", dir, "错误：", err)
		}
	}

	// 持有锁，准备应用配置修改
	h.rwm.Lock()
	defer h.rwm.Unlock()

	// 应用新配置
	if h.watcher != nil {
		h.watcher.Close()
	}
	h.watcher = nWatcher
	h.hostss = hostss
	return nil
}
/*
// 创建目录
func mkdir(path string) error {
	if _, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			err := os.MkdirAll(path, 0711)
			if err != nil {
				return err
			}
		} else {
			return err
		}
	}
	return nil
}
*/

func PathJoin(bashPath, filePath string) string {
	if filepath.IsAbs(filePath) {
		return filePath
	}
	return filepath.Join(bashPath, filePath)
}

// 载入 hosts 文件
// 不处理泛解析及正则解析
// 由于不知道之后由什么处理，并不会进行小写转换。
// 不会检查 ip 格式是不是合法，计划不合法的ip 当作 CNAME 处理。
func LoadHostsFile(fpath string) (d map[string][]string, err error) {
	f, err := os.Open(fpath)
	if err != nil {
		return nil, err
	}

	return LoadHostsStream(f)
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

// 更新循环
// 处理 hosts 本地文件更新 及 http 定期更新。
func (h*hostsDns)loop() {
	wg := sync.WaitGroup{}

	wg.Add(2)
	go func() {
		h.loop_http()
		wg.Done()
	}()
	go func() {
		h.loop_watcher()
		wg.Done()
	}()

	wg.Wait()
}

// 是否已退出
// 是线程安全的，需要小心死锁
func (h*hostsDns)isExited() bool {
	h.rwm.RLock()
	defer h.rwm.RUnlock()
	return h.exited
}

// 监听 hosts 文件修改
func (h*hostsDns)loop_watcher() {
	// 注意，每次更新配置时会关闭上一个 Watcher 并创建一个新的
	// 通过 h.exited 来确认是退出还是更新配置。

	for {
		if h.isExited() {
			return
		}

		h.rwm.RLock()
		watcher := h.watcher
		h.rwm.RUnlock()

		for event := range watcher.Events {
			// 不管是创建、编辑还是重命名，直接匹配路径，发现路径正确就直接重新加载。

			path := "" // 需要更新的路径

			// 检查是否匹配 hosts 文件
			func() {
				h.rwm.RLock()
				defer h.rwm.RUnlock()

				// 循环检查更改的文件是否匹配 hosts 文件
				for _, hosts := range h.hostss {
					if hosts.Local == true && strings.ToLower(hosts.Path) == strings.ToLower(event.Name) {
						path = hosts.Path
						break
					}
				}
			}()

			// 实际载入 hosts 文件
			if path != "" {
				d, err := LoadHostsFile(path)
				if err != nil {
					if os.IsNotExist(err) == false {
						log.Printf("[hosts 更新] 打开文件 %s 失败：%v", path, err)
					}
					break
				}

				func() {
					h.rwm.Lock()
					defer h.rwm.Unlock()
					for _, hosts := range h.hostss {
						if hosts.Local == true && strings.ToLower(hosts.Path) == strings.ToLower(event.Name) {
							hosts.data = d
							hosts.utime = time.Now() //虽然目前没看到更新本地文件的 utime 有什么用...
							log.Printf("[hosts 更新]重新载入 hosts文 件(%v)", hosts.Path)
						}
					}
				}()
			}
		}
	}
}

// 循环更新 http、https hosts文件
func (h*hostsDns)loop_http() {
	for {
		if h.isExited() {
			return
		}

		var checkInterval time.Duration
		now := time.Now()
		// 临时保存需要更新数据
		// 全部下载完毕后再重新更新
		data := make(map[string]map[string][]string)

		// 记录需要更新的 url，尽量短的使用写锁。
		func() {
			h.rwm.RLock()
			defer h.rwm.RUnlock()
			checkInterval = h.checkInterval

			for _, hosts := range h.hostss {
				if hosts.Local == false && now.After(hosts.utime) {
					// 记录
					data[hosts.Path] = nil
				}
			}
		}()


		if len(data) > 0 {
			// 下载hosts文件

			for url, _ := range data {
				// 下载 hosts
				log.Printf("[hosts 更新]开始下载 hosts 文件(%v) ...", url)

				//TODO: 下载 hosts 文件时使用代理
				r, err := http.Get(url)
				if err != nil {
					log.Printf("[hosts 更新]下载 hosts 文件(%v) 失败，错误：%s", url, err)
					continue
				}

				// 解析
				log.Printf("[hosts 更新]开始解析 hosts 文件(%v) ...", url)
				d, err := LoadHostsStream(r.Body)
				if err != nil {
					log.Printf("[hosts 更新]解析 hosts 文件(%v) 失败，错误：%s", url, err)
					continue
				}

				data[url] = d
			}

			// 更新 hosts
			func() {
				h.rwm.Lock()
				defer h.rwm.Unlock()

				for _, hosts := range h.hostss {
					if hosts.Local == false {
						v, ok := data[hosts.Path]
						if ok {
							hosts.data = v
							hosts.utime = now.Add(hosts.updateInterval)
							log.Printf("[hosts 更新] hosts 文件(%v)更新成功。", hosts.Path)
						}
					}
				}
			}()
		}

		time.Sleep(1 * checkInterval)
	}
}

func (h*hostsDns)Close() {
	h.rwm.Lock()
	defer h.rwm.Unlock()

	h.exited = true
	if h.watcher != nil {
		h.watcher.Close()
	}
}
