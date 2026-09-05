package scanners

import (
	"bytes"
	"sync"
)

const maxCommandOutputBytes = 4 << 20 // 4 MiB per scanner invocation

type cappedBuffer struct {
	buf   bytes.Buffer
	limit int
	mu    sync.Mutex
}

func (b *cappedBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	remaining := b.limit - b.buf.Len()
	if remaining <= 0 {
		return len(p), nil
	}
	if len(p) > remaining {
		p = p[:remaining]
	}
	return b.buf.Write(p)
}

func (b *cappedBuffer) Bytes() []byte {
	b.mu.Lock()
	defer b.mu.Unlock()
	return append([]byte(nil), b.buf.Bytes()...)
}
