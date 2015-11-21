package netchan
import (
	"testing"
	"fmt"
	"time"
)

func Test1(t *testing.T) {
	q := NewDnsQuery("www.163.com")
	defer q.Stop()

	for v := range q.RecordChan {
		fmt.Println(*v)
	}

	time.Sleep(100*time.Second)
}

