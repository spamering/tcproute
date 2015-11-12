package main

type UpStream interface {

}



// 上级线路

// 分为 socks 代理、http 代理、https 代理、本地直连 等类型线路

// 好吧，并不需要做太多的处理，还是和 python 版本一样即可。

// 同时存在 简单多路分发线路、http 过滤器线路、https 证书验证器 线路等过滤器线路用来提供特殊功能。

// 简单多路分发器会尝试使用多个线路同时连接，最后使用最快建立连接的线路。

//


// 可以提供



// 表示封装好的 net 模块
// 完全模仿 net 模块实现，所有数据都会通过代理中转。
// 不支持的方法会返回错误。
type ProxyClientNet interface {

}


