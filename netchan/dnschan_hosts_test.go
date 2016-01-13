package netchan

import (
	"testing"
	"time"
	"os"
)

func TestHosts(t *testing.T) {
	os.Remove("hosts-test.txt")

	hostss := make([]*DnschanHostsConfigHosts, 0)
	localHosts := DnschanHostsConfigHosts{
		Path:"hosts-test.txt",
		UpdateInterval:"",
		Credit:0,
		Type:"base",
	}
	httpHosts := DnschanHostsConfigHosts{
		Path:"https://raw.githubusercontent.com/racaljk/hosts/f2699c8652740a6ed000100b505a01a9a0c0730f/hosts",
		UpdateInterval:"60s",
		Credit:0,
		Type:"base",
	}
	hostss = append(hostss, &localHosts)
	hostss = append(hostss, &httpHosts)

	h, err := NewHostsDns(&DnschanHostsConfig{BashPath:"./",
		Hostss:hostss,
		CheckInterval:60 * time.Second,
	})
	if err != nil {
		t.Fatal(err)
	}
	defer h.Close()

	exitChan := make(chan int)
	recordChan := make(chan *DnsRecord)
	ok := false

	time.Sleep(3 * time.Second)

	recordChan = make(chan *DnsRecord)
	go func() {
		h.query("0.docs.google.com", recordChan, exitChan)
		close(recordChan)
	}()

	for r := range recordChan {
		if r.Credit == 0 && r.Ip == "203.208.46.200" {
			ok = true
		}
	}
	if ok == false {
		t.Error("1查询错误")
	}

	// 更改为新地址
	httpHosts.Credit = -123
	httpHosts.Path = "https://raw.githubusercontent.com/racaljk/hosts/47295268db605b4cb85fd9cbe8a88fc4b3536431/hosts"
	h.Config(&DnschanHostsConfig{BashPath:"./",
		Hostss:hostss,
		CheckInterval:60 * time.Second,
	})


	time.Sleep(3 * time.Second)

	recordChan = make(chan *DnsRecord)
	go func() {
		h.query("0.docs.google.com", recordChan, exitChan)
		close(recordChan)
	}()
	ok = false
	for r := range recordChan {
		if r.Credit == -123 && r.Ip == "216.239.38.125" {
			ok = true
		}
	}
	if ok == false {
		t.Error("2查询错误")
	}



	/*
		for _, hosts := range h.hostss {
			if hosts.Path == "https://raw.githubusercontent.com/racaljk/hosts/47295268db605b4cb85fd9cbe8a88fc4b3536431/hosts" {
				hosts.updateInterval = 100 * time.Second
				hosts.utime = time.Now().Add(100 * time.Second)
			}
		}*/

	//测试 hosts 文件
	f, err := os.Create("hosts-test.txt")
	if _, err := f.Write([]byte("   \r\n #000 \r\n 1.2.3.4  aaa.com ")); err != nil {
		t.Fatal(err)
	}
	f.Close()

	time.Sleep(2 * time.Second)

	recordChan = make(chan *DnsRecord)
	go func() {
		h.query("aaa.com", recordChan, exitChan)
		close(recordChan)
	}()
	ok = false
	for r := range recordChan {
		if r.Credit == 0 && r.Ip == "1.2.3.4" {
			ok = true
		}
	}
	if ok == false {
		t.Error("3查询错误")
	}

	os.Remove("hosts-test.txt")

}
