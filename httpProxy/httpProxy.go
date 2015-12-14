package main
import (
	"net/http"
	"time"
	"log"
	"net"
	"io"
	"fmt"
	"sync/atomic"
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


func main() {

	var addr = flag.String("addr",":8080","绑定地址，例子： 0.0.0.0:8080 、:8080")
	flag.Parse()

	s := &http.Server{
		Addr:           *addr,
		Handler:        &myHandler{},
		ReadTimeout:    60 * time.Second,
		WriteTimeout:   60 * time.Second,
		MaxHeaderBytes: 1 << 20,
	}
	log.Fatal(s.ListenAndServe())
}