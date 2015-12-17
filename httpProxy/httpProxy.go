package main
import (
	"net/http"
	"time"
	"log"
	"net"
	"io"
	"fmt"
	"sync/atomic"
	"github.com/inconshreveable/go-vhost"

	"crypto/tls"
	"github.com/golang/glog"
	"bufio"
	"sync"
	"strings"
	"flag"
)

/*
http 代理服务器
proxylient 专用，只支持 CCONNECT 命令。
由于前端有nginx统一处理ssl，这里就不支持 ssl 了。
*/

const forwardBufSize = 8192

type myHandler struct {

}


// 转发计数
// 使用 atomic 实现原子操作
type forwardCount struct {
	send, recv uint64
}

type Flusher interface {
	Flush() error
}

func forwardConn(sRW io.ReadWriter, sConn net.Conn, oConn net.Conn, timeout time.Duration, count *forwardCount) error {
	errChan := make(chan error, 10)

	go _forwardConn(sRW, sConn, oConn, oConn, timeout, errChan, &count.send)
	go _forwardConn(oConn, oConn, sRW, sConn, timeout, errChan, &count.recv)

	return <-errChan
}

func _forwardConn(sR io.Reader, sConn net.Conn, oW io.Writer, oConn net.Conn, timeout time.Duration, errChan chan error, count *uint64) {
	buf := make([]byte, forwardBufSize)
	for {
		sConn.SetDeadline(time.Now().Add(timeout))
		oConn.SetDeadline(time.Now().Add(timeout))
		// 虽然存在 WriteTo 等方法，但是由于无法刷新超时时间，所以还是需要使用标准的 Read、Write。

		if n, err := sR.Read(buf[:forwardBufSize]); err != nil {
			if err == io.EOF {
				errChan <- err
			}else {
				errChan <- fmt.Errorf("转发读错误：%v", err)
			}
			return
		}else {
			buf = buf[:n]
		}
		fmt.Println("收到：", string(buf))

		wbuf := buf
		for {
			if len(wbuf) == 0 {
				break
			}

			if n, err := oW.Write(wbuf); err != nil {
				if err == io.EOF {
					errChan <- err
				}else {
					errChan <- fmt.Errorf("转发写错误：%v", err)
				}
				return
			} else {
				wbuf = wbuf[n:]
			}
		}
		if v, ok := oW.(Flusher); ok {
			v.Flush()
		}

		// 记录转发计数
		atomic.AddUint64(count, uint64(len(buf)))
	}
}

var HTTP_200 = []byte("HTTP/1.1 200 Connection Established\r\n\r\n")
func (h *myHandler)ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method == "CONNECT" {

		host := r.URL.Host

		remoteConn, err := net.DialTimeout("tcp", host, 5 * time.Second)
		if err != nil {
			http.Error(w, "Bad Gateway", http.StatusBadGateway)
			return
		}

		hj, ok := w.(http.Hijacker)
		if !ok {
			http.Error(w, "webserver doesn't support hijacking", http.StatusInternalServerError)
			return
		}
		conn, bufrw, err := hj.Hijack()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		// Don't forget to close the connection:
		defer conn.Close()
		bufrw.Write(HTTP_200)
		bufrw.Flush()

		count := forwardCount{}
		if err := forwardConn(bufrw, conn, remoteConn, 60 * time.Second, &count); err != nil {
			log.Println(err)
		}

	} else {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}
}

// http服务器
type HttpSrever struct {
	httpAddr string

}

func NewHttpServer(httpAddr string) (s*HttpSrever, err error) {
	return &HttpSrever{httpAddr}, nil
}

type Body struct {
	io.Reader
}

func (b Body)Close() error {
	return nil
}

func (s*HttpSrever)Error(w io.Writer, StatusCode int, body string) {
	r := http.Response{StatusCode:StatusCode, Body:Body{strings.NewReader(body)}}
	r.Write(w)
}


// 处理http请求
func (s*HttpSrever)HandlerHttp(conn net.Conn) {
	defer conn.Close()

	rb := bufio.NewReader(conn)
	r, err := http.ReadRequest(rb)
	if err != nil {
		s.Error(conn, http.StatusBadRequest, "Bad Request")
	}

	if r.Method == "CONNECT" {
		host := r.URL.Host

		remoteConn, err := net.DialTimeout("tcp", host, 5 * time.Second)
		if err != nil {
			s.Error(conn, http.StatusBadGateway, "Bad Gateway")
			return
		}

		// Don't forget to close the connection:
		defer remoteConn.Close()
		conn.Write(HTTP_200)

		//TODO： 这里需要设置下超时机制。
		go io.Copy(remoteConn, rb)
		io.Copy(conn, remoteConn)
	} else {
		s.Error(conn, http.StatusBadRequest, "Bad Request")
		return
	}
}


func (s*HttpSrever)ListenAndServe() error {
	l, err := net.Listen("tcp", s.httpAddr)
	if err != nil {
		return err
	}
	for {
		c, err := l.Accept()
		if err != nil {
			if ne, ok := err.(net.Error); ok && ne.Temporary() {
				time.Sleep(5 * time.Millisecond)
				continue
			}
			return err
		}
		go s.HandlerHttp(c)
	}
}


type TlsServer struct {
	httpsAddr, httpsDomain, forwardAddr string
	http                                *HttpSrever
	tlsConfig                           *tls.Config

}

// 处理 tls 请求
func (s*TlsServer)HandlerTls(conn net.Conn) {
	c, err := vhost.TLS(conn)

	if err != nil || c.Host() != s.httpsDomain {
		fmt.Println(err)
		fmt.Println(c.Host())
		// 不匹配，直接转发
		defer conn.Close()

		remoteConn, err := net.Dial("tcp", s.forwardAddr)
		if err != nil {
			glog.Warning(fmt.Printf("[ERR] dial(\"tcp\",%v):%v", s.forwardAddr, err))
			return
		}
		defer remoteConn.Close()

		go io.Copy(conn, remoteConn)
		io.Copy(remoteConn, conn)
	}else {
		s.http.HandlerHttp(tls.Server(conn, s.tlsConfig))
	}
}

func NewTlsServer(httpsAddr, httpsDomain, forwardAddr, httpsCertFile, httpsKeyFile string, httpServer*HttpSrever) (s*TlsServer, err error) {
	cert, err := tls.LoadX509KeyPair(httpsCertFile, httpsKeyFile)
	if err != nil {
		return nil, err
	}
	tlsConfig := tls.Config{Certificates:[]tls.Certificate{cert}}

	s = &TlsServer{httpsAddr:httpsAddr, httpsDomain:httpsDomain, forwardAddr:forwardAddr, http:httpServer, tlsConfig:&tlsConfig}

	return
}

func (s*TlsServer)ListenAndServe() error {
	l, err := net.Listen("tcp", s.httpsAddr)
	if err != nil {
		return err
	}
	for {
		c, err := l.Accept()
		if err != nil {
			if ne, ok := err.(net.Error); ok && ne.Temporary() {
				time.Sleep(5 * time.Millisecond)
				continue
			}
			return err
		}
		go s.HandlerTls(c)
	}
}



func main() {

	//var addr = flag.String("addr",":8080","绑定地址，例子： 0.0.0.0:8080 、:8080")


	httpAddr := flag.String("httpAddr", "", "http 代理绑定地址，例子：0.0.0.0:8080、:8080")
	httpsAddr := flag.String("httpsAddr", "", "https 代理绑定地址，例子：0.0.0.0:443、:443")
	httpsForwardAddr := flag.String("httpsForwardAddr", "", "https 不匹配httpsDomain时转发地址，例子：127.0.0.1:4443")
	httpsDomain := flag.String("httpsDomain", "", "https 代理域名，例子：proxy.com")
	httpsCertFile := flag.String("httpsCertFile", "tls.crt", "https 证书，例子：tls.crt")
	httpsKey := flag.String("httpsKeyFile", "tls.key", "https 证书，例子：tls.key")

	flag.Parse()

	httpServer, err := NewHttpServer(*httpAddr)
	if err != nil {
		panic(err)
	}


	wg := sync.WaitGroup{}

	if *httpAddr != "" {
		wg.Add(1)
		go func() {
			httpServer.ListenAndServe()
			wg.Done()
		}()
	}


	if *httpsAddr != "" {
		wg.Add(1)
		go func() {
			httpsServer, err := NewTlsServer(*httpsAddr, *httpsDomain, *httpsForwardAddr, *httpsCertFile, *httpsKey, httpServer)
			if err != nil {
				panic(err)
			}

			httpsServer.ListenAndServe()
			wg.Done()
		}()
	}

	wg.Wait()
}
