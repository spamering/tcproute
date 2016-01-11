package domains
import (
	"strings"
	"github.com/golang/groupcache/lru"
	"sync"
	"regexp"
	"fmt"
)

/*
域匹配功能


域名匹配策略也需要独立抽取出来了，正好同时用在 hosts 里面。
域名匹配策略有hosts标准匹配、字符串尾匹配、泛解析匹配及正则匹配。

标准匹配直接使用map来做。
尾匹配通过加.然后循环比较，泛解析、正则都转换为正则。为了性能在加个 lru 缓存。

未匹配虽然可以通过分割后通过 map 做，但是感觉性能要求没这么高，而且有缓存，直接循环处理。

*/

type DomainType int

const (
	Base DomainType = iota  //基本匹配，必须完全一致才匹配
	Suffix                  // 后缀匹配 ， abc.com 匹配 www.abc.com 、123.abc.com、123.456.abc.com 及 abc.com
	Pan                     // 泛解析 ，处理 * 及 ?
	Regex                 //正则，正则表达式
)

type UserData interface{}

type Domains struct {
	cacheSize     int
	cache         *lru.Cache               // 域名:[]domainRecord

	rwm           sync.RWMutex

	baseDomains   map[string][]UserData
	suffixDomains map[string][]UserData
	panDomains    map[string]*domainRecord //域名:域记录
	regexDomains  map[string]*domainRecord //正则文本:域记录
}

// 对内单个域
type domainRecord struct {
	domain     string
	domainType DomainType
	userdatas  []UserData
	regexp     *regexp.Regexp
}


// 对外单个域名
// 外部程序不要修改值。
type Domain struct {
	Userdatas []UserData
}


func NewDomains(cacheSize int) *Domains {
	d := Domains{
		cacheSize:cacheSize,
		cache:lru.New(cacheSize),
		baseDomains:make(map[string][]UserData),
		suffixDomains:make(map[string][]UserData),
		panDomains:make(map[string]*domainRecord),
		regexDomains:make(map[string]*domainRecord),
	}
	return &d
}

// 追加新的域名
func (d*Domains)Add(domain string, domainType DomainType, userdata UserData) error {
	d.rwm.Lock()
	defer d.rwm.Unlock()

	switch domainType {
	case Base:
		d.baseDomains[domain] = append(d.baseDomains[domain], userdata)
		d.cache.Remove(domain)
	case Suffix:
		d.suffixDomains[domain] = append(d.suffixDomains[domain], userdata)
		d.cache = lru.New(d.cacheSize)
	case Pan:
		panDomain := d.panDomains[domain]

		if panDomain == nil {
			quoteDomain := regexp.QuoteMeta(domain)
			regexpDomain := strings.Replace(quoteDomain, `\*`, `[^.]+`, -1)
			regexpDomain = strings.Replace(regexpDomain, `\?`, `[^.]`, -1)
			regexpDomain = fmt.Sprint(`^`, regexpDomain, `$`)

			r, err := regexp.Compile(regexpDomain)
			if err != nil {
				return err
			}
			panDomain = &domainRecord{
				domain:domain,
				domainType:domainType,
				regexp:r,
			}
			d.panDomains[domain] = panDomain
		}

		panDomain.userdatas = append(panDomain.userdatas, userdata)

		d.cache = lru.New(d.cacheSize)
	case Regex:

		regexDomain := d.regexDomains[domain]

		if regexDomain == nil {
			r, err := regexp.Compile(domain)
			if err != nil {
				return err
			}
			regexDomain = &domainRecord{
				domain:domain,
				domainType:domainType,
				regexp:r,
			}
			d.regexDomains[domain] = regexDomain
		}

		regexDomain.userdatas = append(regexDomain.userdatas, userdata)

		d.cache = lru.New(d.cacheSize)
	default:
		return fmt.Errorf("不支持类型")
	}

	return nil

}

// 移除域名
// 使用过滤函数来识别需要删除的内容，返回 true 时表示需要删除
func (d*Domains)Remove(f func(domain string, domainType DomainType, uesrdata UserData) bool) {
	d.RemoveType(Base, f)
	d.RemoveType(Suffix, f)
	d.RemoveType(Pan, f)
	d.RemoveType(Regex, f)
}

// 移除域名
// 使用过滤函数来识别需要删除的内容，返回 true 时表示需要删除
func (d*Domains)RemoveType(domainType DomainType, f func(domain string, domainType DomainType, uesrdata UserData) bool) {
	d.rwm.Lock()
	defer d.rwm.Unlock()

	removeBase := func(domainType DomainType, domains map[string][]UserData) *map[string][]UserData {
		newDomains := make(map[string][]UserData)
		for domain, userdatas := range domains {
			newUserdatas := make([]UserData, 0)
			for _, userdata := range userdatas {
				if f(domain, domainType, userdata) == false {
					newUserdatas = append(newUserdatas, userdata)
				}
			}
			if len(newUserdatas) != 0 {
				newDomains[domain] = newUserdatas
			}
		}
		return &newDomains
	}

	removeRegex := func(domainType DomainType, domains *map[string]*domainRecord) {
		delDomains := make([]string, 0)
		for domain, record := range *domains {
			newUserdatas := make([]UserData, 0)
			for _, userdata := range record.userdatas {
				if f(domain, domainType, userdata) == false {
					newUserdatas = append(newUserdatas, userdata)
				}
			}
			record.userdatas = newUserdatas
			if len(newUserdatas) == 0 {
				delDomains = append(delDomains, domain)
			}
		}
		for _, delDomain := range delDomains {
			delete(*domains, delDomain)
		}
	}

	switch domainType {
	case Base:
		d.baseDomains = *removeBase(domainType, d.baseDomains)
	case Suffix:
		d.suffixDomains = *removeBase(domainType, d.suffixDomains)
	case Pan:
		removeRegex(Pan, &d.panDomains)
	case Regex:
		removeRegex(Pan, &d.regexDomains)
	}

	d.cache = lru.New(d.cacheSize)
}

func (d*Domains)RemoveDomain(domain string, domainType DomainType, f func(domain string, domainType DomainType, uesrdata UserData) bool) error {
	d.rwm.Lock()
	defer d.rwm.Unlock()

	switch domainType {
	case Base:
		oldUserdata, ok := d.baseDomains[domain]
		if ok {
			newUserdata := make([]UserData, 0)
			for _, userdata := range oldUserdata {
				if f(domain, domainType, userdata) == false {
					newUserdata = append(newUserdata, userdata)
				}
			}
			if len(newUserdata) != 0 {
				d.baseDomains[domain] = newUserdata
			}else {
				delete(d.baseDomains, domain)
			}
			d.cache.Remove(domain)
		}

	case Suffix:
		oldUserdata, ok := d.suffixDomains[domain]
		if ok {
			newUserdata := make([]UserData, 0)
			for _, userdata := range oldUserdata {
				if f(domain, domainType, userdata) == false {
					newUserdata = append(newUserdata, userdata)
				}
			}
			if len(newUserdata) != 0 {
				d.suffixDomains[domain] = newUserdata
			}else {
				delete(d.suffixDomains, domain)
			}
			d.cache = lru.New(d.cacheSize)
		}

	case Pan:
		record, ok := d.panDomains[domain]
		if ok {
			newUserdatas := make([]UserData, 0)
			for _, userdata := range record.userdatas {
				if f(domain, domainType, userdata) == false {
					newUserdatas = append(newUserdatas, userdata)
				}
			}
			if len(newUserdatas) != 0 {
				record.userdatas = newUserdatas

			}else {
				delete(d.panDomains, domain)
			}
			d.cache = lru.New(d.cacheSize)
		}

	case Regex:
		record, ok := d.regexDomains[domain]
		if ok {
			newUserdatas := make([]UserData, 0)
			for _, userdata := range record.userdatas {
				if f(domain, domainType, userdata) == false {
					newUserdatas = append(newUserdatas, userdata)
				}
			}
			if len(newUserdatas) != 0 {
				record.userdatas = newUserdatas
			}else {
				delete(d.regexDomains, domain)
			}
			d.cache = lru.New(d.cacheSize)
		}
	default:
		return fmt.Errorf("不支持类型")
	}
	return nil
}

// 查找匹配的域名
func (d*Domains)Find(domain string) (*Domain) {
	res := d.cacheGet(domain)
	if res != nil {
		return res
	}

	func() {
		d.rwm.RLock()
		defer d.rwm.RUnlock()

		res = &Domain{}

		// 基本匹配
		u, ok := d.baseDomains[domain]
		if ok {
			res.Userdatas = append(res.Userdatas, u...)
		}

		//后缀匹配
		u, ok = d.suffixDomains[domain]
		if ok {
			res.Userdatas = append(res.Userdatas, u...)
		}
		for k, v := range d.suffixDomains {
			if len(domain) > len(k) &&
			strings.HasSuffix(domain, k) &&
			domain[len(domain) - len(k) - 1] == '.' {
				res.Userdatas = append(res.Userdatas, v...)
			}
		}

		// 泛解析匹配、正则匹配
		regexMatch := func(record map[string]*domainRecord) {
			for _, v := range record {
				if v.regexp.MatchString(domain) {
					res.Userdatas = append(res.Userdatas, v.userdatas...)
				}
			}
		}
		regexMatch(d.panDomains)
		regexMatch(d.regexDomains)
	}()
	d.cacheSet(domain, res)

	return res
}


func (d*Domains)cacheGet(domain string) (*Domain) {
	d.rwm.Lock()
	defer d.rwm.Unlock()

	v, ok := d.cache.Get(domain)
	if ok {
		return v.(*Domain)
	}
	return nil
}
func (d*Domains)cacheSet(domain string, value*Domain) {
	d.rwm.Lock()
	defer d.rwm.Unlock()

	d.cache.Add(domain, value)
	return
}


