package main
import (
	"testing"
	"os"
	"github.com/koding/multiconfig"
	"time"
)


func TestUpStreamDialClient(t *testing.T) {
	f, err := os.Create("Blacklist.txt")
	if _, err := f.Write([]byte("   \r\n #000 \r\n   xxx.com ")); err != nil {
		t.Fatal(err)
	}
	defer os.Remove("Blacklist.txt")


	f, err = os.Create("Whitelist-config.toml")
	if _, err := f.Write([]byte(`
BasePath="."

[[UpStreams]]
Name="direct"
ProxyUrl="direct://0.0.0.0:0000/"
DnsResolve=true
Credit=0
Sleep=0

[[UpStreams.Whitelist]]
Path="https://raw.githubusercontent.com/renzhn/MEOW/af19d0d3e22d485254bf50cfce4e5ed12b0e445b/doc/sample-config/direct"
UpdateInterval="24h"
Type="Suffix"

[[UpStreams.Blacklist]]
Path="Blacklist.txt"
UpdateInterval="24h"
Type="Suffix"

[[UpStreams]]
Name="http"
ProxyUrl="http://123.123.123.123:8088"
DnsResolve=false
Credit=0
Sleep=80

[[UpStreams.Whitelist]]
Path="https://raw.githubusercontent.com/renzhn/MEOW/af19d0d3e22d485254bf50cfce4e5ed12b0e445b/doc/sample-config/proxy"
UpdateInterval="24h"
Type="Suffix"

`)); err != nil {
		t.Fatal(err)
	}
	f.Close()
	defer os.Remove("Whitelist-config.txt")

	config := ConfigDialClients{}

	m := multiconfig.TOMLLoader{Path:"Whitelist-config.toml", } // supports TOML and JSON

	err = m.Load(&config)
	if err != nil {
		t.Fatal(err)
	}

	clients, err := NewDialClients(&config)
	if err != nil {
		t.Fatal(err)
	}

	// 等待白名单、黑名单载入。
	time.Sleep(5 * time.Second)

	// 测试白名单效果
	if dialclients := clients.Get("www.163.com"); len(dialclients) != 1 || dialclients[0].name != "direct" {
		t.Error(clients)
	}

	// 测试黑名单效果
	if dialclients := clients.Get("www.xxx.com"); len(dialclients) != 1 || dialclients[0].name != "http" {
		t.Error(clients)
	}

	// 测试普通域名效果
	if dialclients := clients.Get("www.9999999.com"); len(dialclients) != 2 {
		t.Error(clients)
	}

}
