package main
import (

	"fmt"
	"net"
	"time"
	"io"
	"bytes"
	"reflect"
	"encoding/binary"
	"github.com/golang/glog"
)

const forwardBufSize = 8192 // 转发缓冲区大小

// socks 服务器
type hSocksServer struct {
	srv *Server
}

type hSocksHandle struct {
	hSocksServer *hSocksServer
	conn         net.Conn
}

func NewSocksHandlerNewer(srv *Server) HandlerNewer {
	return &hSocksServer{srv}
}

// 尝试创建 socks 处理器
func (sev *hSocksServer) New(conn net.Conn) (h Handler, rePre bool, err error) {
	// 读取前几个字节，判断是否为 socks 代理协议。
	// 是则返回 handler
	rePre = true
	b := make([]byte, 1024)

	if n, err := conn.Read(b); err != nil {
		err = fmt.Errorf("读取错误：%v", err)
		return
	}else {
		b = b[:n]
	}

	//TODO: 可以进行更详细的判断，例如长度，前几个值可能的范围。
	if b[0] == 0x04 {
		err = fmt.Errorf("暂时不支持 socks4 协议。")
		return
	}else if b[0] == 0x05 {
		return &hSocksHandle{sev, conn}, true, nil
	}else {
		err = NoHandleError("协议头不正确。")
		return
	}
}

func (h *hSocksHandle)String() string {
	return "Socks5"
}

// 实际处理 Socks 协议
func (h*hSocksHandle)Handle() error {
	// 鉴定 + 接受 CMD 的总允许时间
	h.conn.SetDeadline(time.Now().Add(handlerTimeoutHello))

	b := make([]byte, 1)
	if n, err := h.conn.Read(b); err != nil || n != 1 {
		return fmt.Errorf("协议头读取错误：%v", err)
	}

	if b[0] == 0x05 {
		return h.handleSocks5()
	} else if b[0] == 0x04 {
		return h.handleSocks4()
	} else {
		return NoHandleError("无法识别的协议。")
	}
}


func (h*hSocksHandle)handleSocks5() error {
	conn := h.conn //客户端连接

	b := make([]byte, 1, 1024)

	// 读客户端支持的鉴定方法数量
	if n, err := conn.Read(b); err != nil || n != 1 {
		return fmt.Errorf("协议头读取错误：%v", err)
	}
	nmethods := b[0] // 客户端支持的鉴定方法数量

	// 读鉴定方式
	b = b[:nmethods]
	if _, err := io.ReadFull(conn, b); err != nil {
		return fmt.Errorf("读客户端支持的鉴定方法失败：%v", err)
	}

	//判断是否存在无需鉴定
	if bytes.Contains(byte(0), b) != true {
		return fmt.Errorf("客户端不支持无鉴定，登陆失败。客户端支持的鉴定方式：%v", b)
	}

	// 回应鉴定请求
	if _, err := conn.Write([]byte{0x05, 0}); err != nil {
		return fmt.Errorf("回应鉴定错误：%v", err)
	}

	b = b[:5]
	if _, err := io.ReadFull(conn, b); err != nil {
		return fmt.Errorf("读请求错误：%v", err)
	}
	ver := b[0]
	cmd := b[1]
	//rsv := b[2]
	atyp := b[3]
	domainSize := b[4]
	host := ""
	prot := uint16(0)

	// 读地址
	if atyp == 0x01 || atyp == 0x04 {
		//IPv4 or IPv6
		if atyp == 0x01 {
			b = b[:4]
		} else {
			b = b[:16]
		}
		b[0] = domainSize

		if _, err := io.ReadFull(conn, b[1:]); err != nil {
			return fmt.Errorf("读地址错误：%v", )
		}
		host = net.IP(b).String()
	} else if atyp == 0x03 {
		b = b[:domainSize]
		if _, err := io.ReadFull(conn, b); err != nil {
			return fmt.Errorf("读地址错误：%v", )
		}
		host = string(b)
	}else {
		return fmt.Errorf("不支持的地址格式：atyp=%v", atyp)
	}

	// 读端口
	b = b[:2]
	if _, err := io.ReadFull(conn, b); err != nil {
		return fmt.Errorf("读端口错误：%v", err)
	}
	prot = binary.BigEndian.Uint16(b)

	if ver != 0x05 || cmd != 0x01 {
		conn.Write([]byte{0x05, 0x07, 0x00, 0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00})
		return fmt.Errorf("不支持的命令。ver=%v,cmd=%v", ver, cmd)
	}

	conn.SetDeadline(time.Now().Add(handlerTimeoutForward))

	// 连接目标网站
	upStrrem := h.hSocksServer.srv.upStream
	oConn, err := upStrrem.DialTimeout("tcp", net.JoinHostPort(host, string(prot)), handlerNewTimeout)
	if err != nil {
		conn.SetDeadline(time.Now().Add(handlerTimeoutHello))
		conn.Write([]byte{0x05, 0x04, 0x00, 0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00})
		return
	}

	// 获得远端的 IP 及端口
	rIp := net.IP(make([]byte, 4))
	rPort := make([]byte, 2)
	if v, ok := conn.RemoteAddr().(net.TCPAddr); ok {
		rIp = v.IP
		switch len(rIp) {
		case 4, 16:
		default:
			glog.Warning("未知的IP地址类型，IP：%v", rIp)
			rIp = net.IP(make([]byte, 4))
		}
		binary.BigEndian.PutUint16(rPort, uint16(v.Port))
	}

	// 生成返回的消息
	b = b[:0]
	b = append(b, 0x05, 0x00, 0x00)
	if len(rIp) == 4 {
		b = append(b, 0x01)
	} else {
		b = append(b, 0x04)
	}
	b = append(b, rIp)
	b = append(b, rPort)

	if _, err := conn.Write(b); err != nil {
		return fmt.Errorf("回应客户端命令失败：%v", err)
	}

	return forwardConn(conn, oConn, handlerTimeoutForward)
}

func (h*hSocksHandle)handleSocks4() error {
	return fmt.Errorf("未完成")
}


func forwardConn(sConn, oConn net.Conn, timeout time.Duration) {
	errChan := make(chan error)

	go _forwardConn(sConn, oConn, timeout, errChan)
	go _forwardConn(oConn, sConn, timeout, errChan)

	return <-errChan
}

func _forwardConn(sConn, oConn net.Conn, timeout time.Duration, errChan chan error) {
	buf := make([]byte, forwardBufSize)
	for {
		sConn.SetDeadline(time.Now().Add(timeout))
		oConn.SetDeadline(time.Now().Add(timeout))
		// 虽然存在 WriteTo 等方法，但是由于无法刷新超时时间，所以还是需要使用标准的 Read、Write。

		if n, err := sConn.Read(buf[:forwardBufSize]); err != nil {
			errChan <- err
			return
		}else {
			buf = buf[:n]
		}

		for {
			wbuf := buf
			if len(wbuf) == 0 {
				break
			}

			if n, err := oConn.Write(wbuf); err != nil {
				errChan <- err
				return
			} else {
				wbuf = wbuf[n:]
			}
		}
	}

}