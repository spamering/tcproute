package memstream

import (
	"errors"
	"io"
)

// 内存 Stream
// write 空间不足时会自动增加空间，但是除非关闭，否则不会释放内存。
// 由于完全使用内存存放数据，所以不要存放太多的内容。
// 预期存放很多内容请使用 ioutil.TempFile 。
// 注意：有一些方法的实现依赖于 golang 的底层实现，更换golang版本请进行单元测试后次使用。
type MemStream interface {
	io.Reader
	io.WriterTo
	io.Writer
	io.Seeker
	io.Closer
	// Len 剩余可读内容大小
	Len() int
	// 总数据大小
	Size() int64
	// 删除已读数据
	DeleteRead() error
}

type memStream struct {
	d []byte // len
	o int64
}

func NewMemStream() MemStream {
	return &memStream{make([]byte, 0, 128), 0}
}

// Len 剩余可读内容大小
func (m *memStream) Len() int {
	if m.o >= int64(len(m.d)) {
		return 0
	}
	return int(int64(len(m.d)) - m.o)
}

// 总数据大小
func (m *memStream) Size() int64 { return int64(len(m.d)) }

func (m *memStream) Read(p []byte) (n int, err error) {
	rs := m.Len() //可读数据长度

	if rs == 0 {
		return 0, io.EOF
	}

	n = copy(p, m.d[m.o:])
	m.o += int64(n)
	return
}

func (m *memStream) WriteTo(w io.Writer) (n int64, err error) {
	for {
		if m.Len() == 0 {
			return n, io.EOF
		}
		cn, err := w.Write(m.d[m.o:])
		n += int64(cn)
		m.o += n
		if err != nil {
			return n, err
		}
	}
}

func (m *memStream) Write(p []byte) (n int, err error) {
	od := m.d //保存旧的数据，防止非尾部写
	m.d = append(m.d[:m.o], p...)
	if len(m.d) < len(od) {
		// 补充尾部内容
		//m.d = append(m.d, od[m.o]...)
		// 这个方法依赖与底层实现，每次版本更新需要做单元测试。
		m.d = od
	}
	m.o += int64(len(p))
	return len(p), nil
}

var errWhence = errors.New("Seek: invalid whence")
var errOffset = errors.New("Seek: invalid offset")

func (m *memStream) Seek(offset int64, whence int) (int64, error) {
	var no int64

	switch whence {
	case 0:
		no = 0
	case 1:
		no = m.o
	case 2:
		no = m.Size()
	default:
		return 0, errWhence
	}

	no += offset
	if no < 0 {
		return 0, errOffset
	}
	if no > m.Size() {
		m.d = append(m.d, make([]byte, (no-m.Size()))...)
	}
	m.o = no
	return m.o, nil
}

func (m *memStream) Close() error {
	//TODO: 以后为 []byte 使用 sync.pool
	m.d = make([]byte, 0, 0)
	m.o = 0
	return nil
}

func (m *memStream) DeleteRead() error {
	m.d = m.d[m.o:]
	m.o = 0
	return nil
}
