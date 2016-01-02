package main
import (
	"testing"
	"net"
	"time"
	"bytes"
)

func TestCheckPre(t *testing.T) {

	if CheckPre("tcp", "127.0.0.1:80") != preProtocolHttp {
		t.Errorf("CheckPre")
	}

	if CheckPre("tcp", "127.0.0.1:443") != preProtocolHttps {
		t.Errorf("CheckPre")
	}

	if CheckPre("000", "123.com:443") != preProtocolUnknown {
		t.Errorf("CheckPre")
	}

	if CheckPre("tcp", "123.com:443") != preProtocolUnknown {
		t.Errorf("CheckPre")
	}

	if CheckPre("tcp", "123.com") != preProtocolUnknown {
		t.Errorf("CheckPre")
	}
}

func TestPre(t*testing.T) {
	l, err := net.Listen("tcp", ":15348")
	if err != nil {
		t.Fatal(err)
	}

	client := func(d []byte) {
		c, err := net.DialTimeout("tcp", "127.0.0.1:15348", 1 * time.Second)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := c.Write(d); err != nil {
			t.Fatal(err)
		}

		c.Close()
	}


	d2 := make([]byte, 1024)

	d := []byte("GET / HTTP/1.0\r\nHOST:www.test.com\r\n\r\n")
	go client(d)
	c, err := l.Accept()
	c, address, ok := Pre(c, "123.123.123.123:8888", preProtocolHttp)
	if address != "www.test.com:8888" || ok != true {
		t.Error("地址错误,address:", address)
	}
	if n, err := c.Read(d2); err != nil {
		t.Error("读错误:", err)
	} else if bytes.Equal(d, d2[:n]) != true {
		t.Errorf("读数据不匹配")
	}

	d = []byte("GET / HTTP/1.0\r\nHOST:www.test.com:4444\r\n\r\n")
	go client(d)
	c, err = l.Accept()
	c, address, ok = Pre(c, "123.123.123.123:8888", preProtocolHttp)
	if address != "www.test.com:8888" || ok != true {
		t.Error("地址错误,address:", address)
	}
	if n, err := c.Read(d2); err != nil {
		t.Error("读错误:", err)
	} else if bytes.Equal(d, d2[:n]) != true {
		t.Errorf("读数据不匹配")
	}

	d = []byte("GET / HTTP/1.0\r\nHOST:www.test.com\r\n\r\n")
	go client(d)
	c, err = l.Accept()
	c, address, ok = Pre(c, "123.123.123.123:8888", preProtocolHttp)
	if address != "www.test.com:8888" || ok != true {
		t.Error("地址错误,address:", address)
	}
	if n, err := c.Read(d2); err != nil {
		t.Error("读错误:", err)
	} else if bytes.Equal(d, d2[:n]) != true {
		t.Errorf("读数据不匹配")
	}

	d = []byte("GET / HTTP/1.0\r\n\r\n")
	go client(d)
	c, err = l.Accept()
	c, address, ok = Pre(c, "123.123.123.123:8888", preProtocolHttp)
	if address != "123.123.123.123:8888" || ok != false {
		t.Error("地址错误,address:", address)
	}
	if n, err := c.Read(d2); err != nil {
		t.Error("读错误:", err)
	} else if bytes.Equal(d, d2[:n]) != true {
		t.Errorf("读数据不匹配")
	}
}

func TestIn(t *testing.T) {
	if in(80, []int{80}) == false {
		t.Error("In")
	}
	if in(443, []int{80}) == true {
		t.Error("In")
	}
}