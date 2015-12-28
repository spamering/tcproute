package netchan
import (
	"testing"
)

func Test1(t *testing.T) {
	q := NewDnsQuery("www.163.com")
	defer q.Stop()

	i:=0

	for _ = range q.RecordChan {
		i++
	}

	if i==0{
		t.Fatal("dns")
	}
}

