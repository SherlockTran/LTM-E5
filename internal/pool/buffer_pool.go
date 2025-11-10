package pool

import "sync"

const (
	defaultBufSize = 64 * 1024
)

var byteBufPool = sync.Pool{
	New: func() any {
		b := make([]byte, defaultBufSize)
		return &b
	},
}

func Get() *[]byte {
	return byteBufPool.Get().(*[]byte)
}

func Put(b *[]byte) {
	if b == nil || *b == nil {
		return
	}
	// Keep capacity stable
	(*b) = (*b)[:cap(*b)]
	byteBufPool.Put(b)
}


