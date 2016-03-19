package main
import (

	"fmt"
	"net"
	"time"
	"io"
	"bytes"
	"encoding/binary"
	"strconv"
	"log"
	"sync/atomic"
)

const forwardBufSize = 8192 // 转发缓冲区大小

// socks 服务器
type hSocksServer struct {
	upStream UpStreamDial
}

type hSocksHandle struct {
	hSocksServer *hSocksServer
	conn         net.Conn
}

func NewSocksHandlerNewer(upStream UpStreamDial) HandlerNewer {
	return &hSocksServer{upStream}
}

// 尝试创建 socks 处理器
func (sev *hSocksServer) New(conn net.Conn) (h Handler, rePre bool, err error) {
	// 读取前几个字节，判断是否为 socks 代理协议。
	// 是则返回 handler
	rePre = true
	b := make([]byte, 1024)

	if n, cerr := conn.Read(b); cerr != nil {
		err = fmt.Errorf("读取错误：%v", cerr)
		fmt.Println(err)
		fmt.Println(b[:100])
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
	if bytes.Contains(b, []byte{0}) != true {
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

	rAddr := net.JoinHostPort(host, strconv.FormatUint(uint64(prot), 10))

	// 检查是否需要进行预处理
	// 目前的预处理只在浏览器进行了本地DNS解析时强制转换为代理进行DNS解析。
	preProtocol := CheckPre("tcp", rAddr)
	if preProtocol != preProtocolUnknown {
		// 如果需要预处理，那么会先给客户端发出已连接成功的回应来尝试获得客户端发出的请求头进行预处理
		if _, err := conn.Write([]byte{0x05, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}); err != nil {
			return fmt.Errorf("回应客户端命令失败：%v", err)
		}
		nConn, nAddress, ok := Pre(conn, rAddr, preProtocol)
		if ok {
			log.Printf("[强制代理服务器DNS解析] 对目的地址是 %v 的请求进行了重定向，新目标：%v", rAddr, nAddress)
		}
		conn = nConn
		rAddr = nAddress
	}

	// 连接目标网站
	upStrrem := h.hSocksServer.upStream
	oConn, oConnErrorReporting, err := upStrrem.DialTimeout("tcp", rAddr, handlerNewTimeout)
	if err != nil {
		conn.SetDeadline(time.Now().Add(handlerTimeoutHello))
		if preProtocol == preProtocolUnknown {
			conn.Write([]byte{0x05, 0x04, 0x00, 0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00})
		}
		return fmt.Errorf("无法连接目标网站( %v )，详细错误：%v", rAddr, err)
	}
	defer oConn.Close()

	// 如果进行了预处理就不用在发出已经建立连接的回应了，预处理时已经发出了。
	if preProtocol == preProtocolUnknown {
		// 获得远端的 IP 及端口
		rIp := make([]byte, 4, 16)
		rPort := make([]byte, 2)
		if v, ok := conn.RemoteAddr().(*net.TCPAddr); ok {
			if ip4 := v.IP.To4(); ip4 != nil {
				rIp = []byte(ip4)
			}else {
				rIp = []byte(v.IP)
			}
			switch len(rIp) {
			case 4, 16:
			default:
				log.Printf("未知的IP地址类型，IP：%v", rIp)
				rIp = make([]byte, 4)
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
		b = append(b, rIp...)
		b = append(b, rPort...)

		if _, err := conn.Write(b); err != nil {
			return fmt.Errorf("回应客户端命令失败：%v", err)
		}
	}

	fCount := forwardCount{} //转发计数


	startTime := time.Now()
	err = forwardConn(conn, oConn, handlerTimeoutForward, &fCount)
	endTime := time.Now()

	// 识别异常状态
	// 连接被重置、未收到任何数据
	if oConnErrorReporting != nil {
		connTime := endTime.Sub(startTime)
		lsend := atomic.LoadUint64(&fCount.send)
		lrecv := atomic.LoadUint64(&fCount.recv)

		// 连接建立时间小于60秒，并且未收到任何数据
		if connTime < 60 * time.Second && lrecv == 0 && lsend > 50 {
			log.Printf("记录代理转发错误：连接建立时间 %v 小于60秒，已发出 %v 数据，但未收到任何数据。详细信息，%v，错误：%v\r\n",
				float64(connTime)/float64(time.Second),lsend,oConnErrorReporting.GetInfo(),err)
			oConnErrorReporting.Report(ErrConnTypeRead0)
		}

		if connTime < 1 * time.Second && lrecv < 1024 && prot == 443 && lsend > 50 {
			log.Printf("记录代理转发错误：https 连接建立时间 %v 小于1秒，已发出 %v 字节数据，收到 %v 字节数据。详细信息，%v，错误：%v\r\n",
				float64(connTime)/float64(time.Second),lsend,lrecv,oConnErrorReporting.GetInfo(),err)
			oConnErrorReporting.Report(ErrConnTypeRead0)
		}
	}

	//fmt.Printf("接收：%v ,发送：%v\r\n", fCount.recv, fCount.send)

	return err
}

func (h*hSocksHandle)handleSocks4() error {
	return fmt.Errorf("未完成")
}


