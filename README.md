# TcpRoute2

[![Build Status](https://travis-ci.org/GameXG/TcpRoute2.svg)](https://travis-ci.org/GameXG/TcpRoute2) [![release](https://img.shields.io/github/release/gamexg/tcproute2.svg)](https://github.com/GameXG/TcpRoute2/releases) [![GitHub license](https://img.shields.io/badge/license-MIT-blue.svg)](https://raw.githubusercontent.com/GameXG/TcpRoute2/master/LICENSE)  [![platform](https://img.shields.io/badge/platform-%20windows%20%7C%20linux%20%7C%20freebsd%20%7C%20darwin%20-yellow.svg)](https://github.com/GameXG/TcpRoute2/releases)


TcpRoute , TCP 层的路由器。对于 TCP 连接自动从多个线路(允许任意嵌套)、多个域名解析结果中选择最优线路。

TcpRoute 使用激进的选路策略，对 DNS 解析获得的多个IP同时尝试连接，同时使用多个线路(代理)进行连接，最终使用最快建立的连接。支持 TcpRoute 级别 Hosts 文件，支持黑白名单。提供代理、hosts 信誉度功能，只通过不安全的代理转发 https 等加密连接，提高安全性。当配合 redsocks、Proxifier 作为全局代理时可以启动“强制TcpRoute Dns解析”，强制将浏览器本地 DNS 解析改为代理服务器进行DNS解析来更好的优化网络连接，避免 Dns 污染造成的网络故障。

增加了反运营商 http 劫持功能，有两种方式，简易拆包反劫持及ttl反劫持。

通过 socks5 代理协议对外提供服务。

代理功能拆分成了独立的库，详细代理url格式及选项请参见 [ProxyClient](https://github.com/GameXG/ProxyClient)，目前支持直连、socks4、socks4a、socks5、http、https、ss 代理线路。其中 socks5 支持用户名、密码认证，http、https 支持用户名、密码基本认证。

## 安装

在 [releases](https://github.com/GameXG/TcpRoute2/releases) 有各个系统的 zip 包。根据系统下载对应的 zip 文件。解压后复制 config.toml.example 为 config.toml ，并根据 toml 内说明配置好上游代理即可。

Windows 下有图形界面，TcpRoute2-windows-gui-386.zip、TcpRoute2-windows-gui-amd64.zip 即带图形界面的版本。

## 配置

默认使用当前目录下的 config.toml 文件。

``` toml

# TcpRoute2 配置文件
# https://github.com/GameXG/TcpRoute2
# 为 TOML 格式，格式说明：https://segmentfault.com/a/1190000000477752

# TcpRoute 监听地址
# 目前只对外提供 socks5 协议。
#
# addr = "127.0.0.1:7070"
# 默认值，表示监听 127.0.0.1 的 7070 端口，仅本机使用 TcpRoute 时建议这样配置。
# 将浏览器代理设置为 socks5 127.0.0.1:7070 即可使用 TcpRoute 代理访问网络。
#
# addr = ":7070"
# 监听所有 ip 地址的 7070 端口，允许其他计算机使用 TcpRoute 访问网络时建议这样配置。
# 将浏览器代理设置为 socks5 TcpRoute计算机IP:7070 即可使用 TcpRoute 代理访问网络。

addr="127.0.0.1:7070"


####################
# 客户端dns解析纠正功能
####################
# 当发现浏览器等客户端进行了本地dns解析时本功能将强制转换为 TcpRoute 进行dns解析。
# 使用 redsocks、Proxifier 实现全局代理时，应用程序会进行本地dns解析，启用这个功能将强制为代理进行dns解析。
# 开启这个功能将避免应用程序本地dns解析时无法避免 dns 污染的问题，同时代理负责DNS解析也能更好的优化网络访问。
#
# chrome 默认是远端dns解析，当不使用 redsocks、Proxifier 时不需要这个功能。
# firefox 很早之前默认是本地 dns 解析，不过可以修改为远端dns解析。目前是什么情况就不知道了。
#
# https 协议下 TcpRoute 是通过 SNI 功能来获得的目标网站域名。
# 因为 WinXP 系统下 IE 所有版本都不支持 SNI 功能，所以 windows xp IE 下 https 强制远端解析功能无效。
#
# 例子：
# PreHttpPorts=[80,]
# PreHttpsPorts=[443,]
# 这个是默认值，对 80 端口的 http 请求启用，对 443 端口的 tls 连接启用。
#
# PreHttpPorts=[0,]
# PreHttpsPorts=[0,]
# 关闭这个功能
#
# 原理：
# TcpRoute 接收到目的地址是域名的请求将不执行“客户端dns解析纠正功能”，
# 但当目的地址是 ip 时，将会读取客户端发出的请求，http 读取 hosts 字段获得域名，https 通过 SNI 功能获得域名。
# 之后将目标网站ip替换为域名，再执行转发操作。


####################
# 线路
####################
#
# TcpRoute 将根据一定的策略使用这里指定的线路(上游代理)将收到的请求请求转发出去。
#
# 目前的策略是 TcpPing ，即同时使用多个线路建立连接，最终使用最快建立连接的线路处理请求。
# 当某条线路访问某网站出现异常(响应超时、连接重置等)时将会被记录下来，下次访问同一网站时将跳过这个线路。
# 允许通过黑白名单指定每个线路允许、拒绝访问指定的网站。
#
# 目前上游代理支持 直连、http、https、socks4、socks4a、socks5 及 ss 协议，其中 http、https、socks5、ss 支持密码认证。
# 注意：直连也必须手工指定，当不指定时将不会使用直连转发请求。
#
##########
# 线路(上游代理)配置说明
##########
# [[UpStreams]]
#
# Name="direct"
# 线路名字，主要是日志使用。默认值为 ProxyUrl 项的内容。
#
#
# ProxyUrl="direct://0.0.0.0:0000"
# 线路(上游代理) URL
# 提供代理的类型、地址、用户认证方式等信息。
# 默认值为："direct://0.0.0.0:0000"
#
# 支持 直连、http、https、socks4、socks4a、socks5 及 ss 协议，其中 http、https、socks5、ss 支持密码认证。
# 允许多层嵌套代理。代理部分已经拆分成了独立的库，详细配置信息可以到 https://github.com/GameXG/ProxyClient 参看。
#
# 可以通过参数指定一些特殊选项，例如，https 代理是否验证服务器 tls 证书。
# 参数格式为：?参数名1=参数值1&参数名2=参数值2
# 例如：https://123.123.123.123:8088?insecureskipverify=true
#     全体协议可选参数： upProxy=http://145.2.1.3:8080 用于指定代理的上层代理，即代理嵌套。默认值：direct://0.0.0.0:0000
#
# 支持的代理协议：
# http 代理 http://123.123.123.123:8088
#     可选功能： 用户认证功能。格式：http://user:password@123.123.123:8080
#     可选参数：standardheader=false true表示 CONNNET 请求包含标准的 Accept、Accept-Encoding、Accept-Language、User-Agent等头。默认值：false
#
# https 代理 https://123.123.123.123:8088
#     可选功能： 用户认证功能，同 http 代理。
#     可选参数：standardheader=false 同上 http 代理
#     可选参数：insecureskipverify=false true表示跳过 https 证书验证。默认false。
#     可选参数：domain=域名 指定https验证证书时使用的域名，默认为 host:port
#
# socks4 代理 socks4://123.123.123.123:5050
#     注意：socks4 协议不支持远端 dns 解析
#
# socks4a 代理 socks4a://123.123.123.123:5050
#
# socks5 代理 socks5://123.123.123.123:5050
#     可选功能：用户认证功能。支持无认证、用户名密码认证，格式同 http 代理。
#
# ss 代理 ss://method:passowd@123.123.123:5050
#
# 直连 direct://0.0.0.0:0000
#     可选参数： LocalAddr=0.0.0.0:0 表示tcp连接绑定的本地ip及端口，默认值 0.0.0.0:0。
#     可选参数： SplitHttp=false true 表示拆分 http 请求(分多个tcp包发送)，可以解决简单的运营商 http 劫持。默认值：false 。
#              原理是：当发现目标地址为 80 端口，发送的内容包含 GET、POST、HTTP、HOST 等关键字时，会将关键字拆分到两个包在发送出去。
#              注意： Web 防火墙类软件、设备可能会重组 HTTP 包，造成拆分无效。目前已知 ESET Smart Security 会造成这个功能无效，即使暂停防火墙也一样无效。
#              G|ET /pa|th H|TTTP/1.0
#              HO|ST:www.aa|dd.com
#     可选参数： sleep=0  建立连接后延迟多少毫秒发送数据，配合 ttl 反劫持系统时建议设置为10置50。默认值 0 。
#
# DnsResolve=true
# 是否执行本地dns解析,只建议直连、socks4 线路设置为 true 。
# 设置为 true 时将由 TcpRoute 进行本地 DNS 解析，目前主要是同时使用本地操作系统dns解析及 TcpRoute hosts dns解析。
# 解析获得多个IP时将会同时建立到多个ip的连接，最终使用最快建立连接的ip。
# 设置为 false 时将由上游代理负责dns解析。建议 http、https、socks4a、socks5、ss代理都设置为 false 。
# 默认值 false
#
#
# Credit=0
# 线路的信誉度
# 代理线路不安全时建议使用这个选项。
# 当信誉度低于 0 时将不会通过这个线路建立明文协议(http、ftp、stmp等)的连接。
# 各协议需要的信誉度：https://github.com/GameXG/TcpRoute2/blob/master/netchan/dialchan_filter.go#L19
# 默认值 0
#
#
# Sleep=80
# 使用本线路前等待的时间(单位毫秒)
# 主要目的是降低上游代理的负担。
# 建议直连线路设置为 0 ，代理线路设置为 80(毫秒) 。
# 国内 baidu、qq tcping一般是30ms，这里设置为80ms(0.08秒)，
# 可以使得大部分国内站点不会尝试通过代理访问，降低上游代理的负担。
# 0.08秒的延迟很低，并且建立连接后会缓存最快连接记录，不会再次延迟，所以不建议删除。
# 当目标网站匹配域名白名单、黑名单，即手工指定线路时，Sleep 参数无效。
# 默认值 0
#
#
# CorrectDelay=0
# 修正延迟
# ss 协议缺陷，ss 协议并不会报告是否连接到了目标网站，所以无法获得真实的建立到到目标网站的耗时。
# 无法获得准确的到目标网站的耗时将使得 tcpping 策略无法准确的评估各个线路的速度，所以增加了这个选项用来手工修正。
# tcpping 策略评估最快建立连接的线路时会以 “建立连接的实际耗时 + CorrectDelay” 进行评估，选出最快建立连接的线路。
# 非 ss 协议不用设置，ss 协议建议设置为50-100。
#
#
######
# 域名白名单
######
# 当前线路的域名白名单（线路级别），白名单内的域名将只从当前线路转发。
# 当一个域名同时存在多个线路白名单内时，将会同时从多个线路尝试建立连接。
# 单个线路可以配置多个白名单，每个白名单一个 [[UpStreams.Whitelist]] 即可。
# 当不存在 [[UpStreams.Whitelist]] 项目时即表示不配置白名单。
# 当目标网站匹配域名白名单、黑名单，即手工指定线路时，线路的 Sleep 参数无效。
#
# [[UpStreams.Whitelist]]
#
# Path="direct.txt"
# 允许本地文件及 http 、https 白名单文件。
# 本地路径是相对路径时，实际路径是相对于 config.toml 文件所在目录。TcpRoute 会检测 hosts 文件修改并自动重新载入。
# http、https 域名文件将按 UpdateInterval 间隔定时更新。
# 不允许单个 [[UpStreams.Whitelist]] 下面出现多个 Path 指定多个白名单文件。
# 多个白名单需要分别放到不同的 [[UpStreams.Whitelist]] 下面。
# 默认值：必填
#
# Path="https://raw.githubusercontent.com/renzhn/MEOW/master/doc/sample-config/direct"
# 感谢 renzhn MEOW 维护的境内网站白名单
#
# Path="https://raw.githubusercontent.com/GameXG/TcpRoute2/master/direct.txt"
# 感谢 puteulanus 整理的 unblock youku 最小国内网站白名单。
#
#
# UpdateInterval="24h"
# 网络 hosts 文件更新间隔
# 最小有效值 1 分钟, 格式为："1.5h" or "2h45m". Valid time units are "ns", "us" (or "µs"), "ms", "s", "m", "h".
# 下载失败时 UpdateInterval 不会生效，将等待 1 分钟重试。
# 默认值： "24h"
#
#
# Type="suffix"
# 域名类型
# 指定域名文件的匹配类型
# base 完整匹配，默认值。即 www.abc.com 只匹配 www.abc.com ，不匹配 aaa.www.abc.com 。
# suffix 后缀匹配。即 abc.com 匹配 abc.com、www.abc.com、aaa.www.abc.com，不匹配 aaaabc.com。
# pan 泛解析匹配。即 *.abc.com 匹配 www.abc.com 、aaa.www.abc.com。不匹配 .abc.com。?.abc.com 匹配 a.abc.com。
# regex 正则匹配。即 ^.+?.com$ 匹配 www.abc.com 、aaa.www.abc.com。注意：完整匹配时不要忘记 ^$ 。
# 默认值："base"
#
######
# 域名黑名单
######
# 同域名白名单
# [[UpStreams.Whitelist]]
# 同白名单。
#
# Path="proxy.txt"
#
# Path="https://raw.githubusercontent.com/renzhn/MEOW/master/doc/sample-config/proxy"
# 感谢 renzhn MEOW 维护的网站黑名单
#
# UpdateInterval="24h"
# Type="suffix"


# 直连线路例子：
# 注意：直连也必须手工指定，当不指定时将不会使用直连转发请求。
[[UpStreams]]
Name="direct"
ProxyUrl="direct://0.0.0.0:0000"
DnsResolve=true
# DnsResolve 表示是否执行本地dns解析，直连线路建议指定为 true。

# 直连线路域名白名单
# 各个线路的白名单、黑名单是独立的，可以通过多个 [[UpStreams.Whitelist]] 指定多个白名单。
[[UpStreams.Whitelist]]
Path="https://raw.githubusercontent.com/GameXG/TcpRoute2/master/direct.txt"
# 感谢 puteulanus 整理的 unblock youku 最小国内网站白名单。
UpdateInterval="24h"
Type="suffix"

# 代理线路例子：
[[UpStreams]]
Name="proxy1"
ProxyUrl="socks5://123.123.123.123:5050"
Credit=0
# Credit 表示代理信誉度，低于0的将不会通过当前线路转发明文协议(http、ftp等)的请求。
Sleep=80
# Sleep表示使用本线路前等待的时间，单位毫秒。
CorrectDelay=0
# CorrectDelay 表示当前线路修正延迟，ss 协议建议设置为 50-100 之间的值，非 ss 协议代理设置为 0。


####################
# Hosts 功能
####################
# 独立与操作系统的 hosts，只对于代理生效。
# 允许通过多个 [[hosts]] 项来同时使用多个 hosts 文件 。
#
# [[Hosts]]
#
# Path="hosts/racaljk_hosts.txt"
# hosts 路径，同域名白名单，允许本地、http、https文件。
# 默认值：必填
#
# Path="https://raw.githubusercontent.com/racaljk/hosts/master/hosts"
# 感谢 https://github.com/racaljk/hosts 项目维护 hosts
#
#
# Type="base"
# hosts 域名类型，同域名白名单。标准的 hosts 文件都是 base 类型。
# 默认值："base"
#
#
# UpdateInterval="24h"
# 网络 hosts 文件更新间隔
# 最小有效值 1 分钟， 格式："1.5h" or "2h45m". Valid time units are "ns", "us" (or "µs"), "ms", "s", "m", "h".
# 下载失败时 UpdateInterval 不会生效，将等待 1 分钟重试。
# 默认值："24h"
#
#
# Credit=-100
# hosts 信誉度
# 同线路信誉信誉度，对于小于 0 的 hosts文件将只用于 https 等自带加密的协议。
# 在某些情况下为了防止 http 明文协议分析阻断连接，建议设置为小于 0 的值。
# 默认值：0

# 一个例子
[[Hosts]]
Path="https://raw.githubusercontent.com/racaljk/hosts/master/hosts"
# 感谢 https://github.com/racaljk/hosts 项目维护 hosts
Credit=-100


```

## 强制代理服务器 DNS 解析功能

redsocks、Proxifier 全局代理及部分应用会执行本地DNS解析，这样将无法很好的执行优化。启用这个功能后 TcpRoute2 将在发现客户端执行了本地 DNS 解析时强制改为代理服务器进行DNS解析来更好的优化网络连接。

解决了路由器通过 redsocks 配置成全局透明代理时无法应对dns污染的问题。

由于 https 协议是通过 SNI 功能来获得的目标网站域名，所以 WinXP 系统下 IE 所有版本都无法使用 https 强制远端解析功能。


## 信誉度功能

增加了代理信誉度、dns信誉度的功能，对于信誉度低的代理将只允许 https 、smtp ssl 等本身支持服务器认证的协议。这样使用不安全的代理也能比较安全。

## Hosts 功能

增加了代理级别的 hosts 文件，支持本地及网络hosts文件。通过hosts即使在不存在上层代理的情况下也可以优化网络访问。hosts 文件同样也有信誉度功能。

感谢 https://github.com/racaljk/hosts 项目维护 hosts 。

## 白名单、黑名单功能

允许指定的域名走指定的线路，指定的域名不走指定的线路。
黑白名单是线路级别的，而不是全局的，每个线路都有自己的黑白名单。

感谢 https://github.com/renzhn/MEOW 维护了国内域名白名单。

## 防运营商 HTTP 劫持功能

### 拆包反劫持功能

通过拆分 http 请求到多个 tcp 包来实现简易http反劫持功能，只能应付简单的http劫持。
通过 SplitHttp 选项开启，默认关闭。
注意：部分杀毒软件、防火墙会重组 http 请求tcp包，造成这个功能无效。

基本不会造成性能损耗。

实现原理：当目标端口是80时，发出的请求一旦包含 GET、POST、HTTP、HOST则会被拆分到多个TCP包发送。

### ttl 反劫持功能

ttl 反劫持是独立的程序，需要单独启动ttl反劫持程序，并将直连线路的 sleep 设置为10至50之间的值。

实现原理：当发现 http 连接建立时，将会发送错误数据、连接重置命令混淆http连接。通过设置 ttl 值使得错误数据、重置命令不会到达目标网站，只会在网络中传递，经过并干扰可能存在的http劫持系统。

sleep 的目的是建立连接后不立刻发送数据，而是等待 ttl 反劫持程序发送混淆内容后再发送实际数据。
一般ttl反劫持程序发送混淆数据的耗时为10-30毫秒，sleep设置为大于这个值即可。

ttl 反劫持程序地址：https://github.com/GameXG/ProxyClient/tree/master/ttl

## 具体细节
* 对 DNS 解析获得的多个IP同时尝试连接，最终使用最快建立的连接。
* 同时使用直连及代理建立连接(可设置延迟)，最终使用最快建立的连接。
* 缓存10分钟上次检测到的最快线路方便以后使用。
* 解析不存在域名获得域名纠错IP，并添加到 IP黑名单
