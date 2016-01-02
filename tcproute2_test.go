package main
import (
	"testing"
	"github.com/gamexg/proxyclient"
	"bytes"
)

// 还未实现全自动测试
func testPre1(t *testing.T) {
	p, err := proxyclient.NewProxyClient("socks5://127.0.0.1:7070")
	if err != nil {
		t.Fatal(err)
	}

	c, err := p.Dial("tcp", "0.0.0.1:80")
	if err != nil {
		t.Fatal(err)
	}

	if _, err := c.Write([]byte("GET / HTTP/1.0\r\nHOST:www.baidu.com\r\n\r\n")); err != nil {
		t.Fatal(err)
	}

	buf := make([]byte, 1024)

	if n, err := c.Read(buf); err != nil {
		t.Fatal(err)
	}else {
		if bytes.Contains(buf[:n], []byte("Content-Type")) == false {
			t.Fatal("失败,数据：",string(buf[:n]))
		}
	}
}