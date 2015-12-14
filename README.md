# TcpRoute2
TCP 层的路由器，自动尽可能的优化 TCP 连接。 golang 重写的 TcpRoute 。

## 信誉度功能

增加了代理信誉度、dns信誉度的功能，对于信誉度低的代理将只允许 https 、smtp ssl 等本身支持服务器认证的协议。

