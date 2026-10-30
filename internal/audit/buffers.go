package audit

import (
	"bytes"
	"sync"
)

var bufPool = sync.Pool{
	New: func() any { return new(bytes.Buffer) },
}

func getBuf() *bytes.Buffer {
	b := bufPool.Get().(*bytes.Buffer)
	b.Reset()
	return b
}

func putBuf(b *bytes.Buffer) {
	if b.Cap() > 64*1024 {
		return
	}
	bufPool.Put(b)
}
