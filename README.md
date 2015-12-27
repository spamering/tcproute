# TcpRoute2

golang 重写的 https://github.com/GameXG/TcpRoute 。

TcpRoute, TCP 层的路由器。对于 TCP 连接自动从多个线路(允许任意嵌套)、多个域名解析结果中选择最优线路。

通过 socks5 代理协议对外提供服务。

通过 https://github.com/GameXG/ProxyClient 实现上层代理支持，目前支持直连、socks4、socks4a、socks5、http、https、ss 代理线路。

## 配置

默认使用当前目录下的 config.toml 文件。

``` toml

# TcpRoute2 配置文件
# https://github.com/GameXG/TcpRoute2

# 监听地址
# 目前只对外提供 socks5 协议
addr="127.0.0.1:7070"


# 上层代理
# 连接网络时会自动选择最快的线路。
# 注意，直连也需要提供，否则不会通过直连访问网络。

[[UpStreams]]
Name="direct"
# 代理地址
# 通过 https://github.com/GameXG/ProxyClient 实现的代理功能
# 目前支持以下格式
# http 代理 http://123.123.123.123:8088
# https 代理 https://123.123.123.123:8088
# socks4 代理 socks4://123.123.123.123:5050  socks4 协议不支持远端 dns 解析
# socks4a 代理 socks4a://123.123.123.123:5050
# socks5 代理 socks5://123.123.123.123:5050?upProxy=http://145.2.1.3:8080
# ss 代理 ss://method:passowd@123.123.123:5050
# 直连 direct://0.0.0.0:0000/?LocalAddr=123.123.123.123:0
ProxyUrl="direct://0.0.0.0:0000"
# 是否执行本地dns解析
DnsResolve=true


[[UpStreams]]
Name="https-proxy.com"
ProxyUrl="https://www.proxy.com:443"
DnsResolve=false
# 线路的信誉度，不会通过信誉度低于0的代理建立明文协议的连接(http、ftp、stmp等)
Credit=0

# 使用本线路前等待的时间(单位毫秒)
# 国内 baidu、qq tcping一般是30ms，这里设置为80ms(0.08秒)。
# 可以使得大部分国内站点不会尝试通过代理访问，降低上游代理的负担。
# 0.08秒的延迟很低，并且建立连接后会缓存最快连接记录，不会再次延迟，所以不建议删除。
Sleep=500

# 修正延迟
# ss 协议并不会报告是否连接到了目标网站，所以无法获得真实的到目标网站的 tcpping。
# 目前只能通过 ss 服务器 tcpping + CorrectDelay 来估算。
# 非 ss 协议不用设置，ss 协议建议设置为50-100.
CorrectDelay=0



```

## 信誉度功能

增加了代理信誉度、dns信誉度的功能，对于信誉度低的代理将只允许 https 、smtp ssl 等本身支持服务器认证的协议。这样即使使用他人的代理也能比较安全了。


## 具体细节
* 对 DNS 解析获得的多个IP同时尝试连接，最终使用最快建立的连接。
* 同时使用直连及代理建立连接(可设置延迟)，最终使用最快建立的连接。
* 缓存10分钟上次检测到的最快线路方便以后使用。
* 解析不存在域名获得域名纠错IP，并添加到 IP黑名单
