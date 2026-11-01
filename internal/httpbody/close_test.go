package httpbody

import (
	"io"
	"strings"
	"testing"
)

type errCloser struct {
	closed bool
	err    error
}

func (c *errCloser) Close() error {
	c.closed = true
	return c.err
}

func TestCloseNilAndError(t *testing.T) {
	Close(nil)
	body := io.NopCloser(strings.NewReader("ok"))
	Close(body)
	c := &errCloser{err: io.ErrClosedPipe}
	Close(c)
	if !c.closed {
		t.Fatal("expected Close to call the closer")
	}
}
