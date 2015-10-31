package memstream

import (
	//"bytes"
	"bytes"
	"io"
	"testing"
)

func Test(t *testing.T) {
	m := NewMemStream()
	ms, ok := m.(*memStream)
	if ok != true {
		t.Error("err:type")
	}
	n, err := m.Write([]byte{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10})
	if n != 11 || err != nil {
		t.Error("err:write")
	}
	if ms.o != 11 || bytes.Equal(ms.d, []byte{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10}) != true {
		t.Error("err:data", ms.d)
	}

	n64, err := m.Seek(-6, 1)
	if n64 != 5 || err != nil {
		t.Error("err:seek 1")
	}

	b := []byte{0}
	n, err = m.Read(b)
	if n != 1 || err != nil || b[0] != 5 {
		t.Error("err:read")
	}

	if m.Size() != 11 {
		t.Error("err:Size")
	}

	if m.Len() != 5 {
		t.Error("Len")
	}

	n64, err = m.Seek(7, 1)
	if n64 != 13 || err != nil {
		t.Error("seek 2")
	}

	b2 := []byte{99, 88, 77}
	n, err = m.Write(b2)
	if n != 3 || err != nil {
		t.Error("write2")
	}

	n64, err = m.Seek(5, 0)
	if n64 != 5 || err != nil {
		t.Error("seek 3")
	}

	m2 := NewMemStream()
	ms2, ok := m2.(*memStream)
	if ok != true {
		t.Error("eee")
	}
	cr := m.Len()
	n64_2, err := m.WriteTo(m2)
	if n64_2 != int64(cr) || err != io.EOF {
		t.Error("write to")
	}

	if bytes.Equal(ms.d[n64:], ms2.d) != true {
		t.Error("write to2")
	}
}
