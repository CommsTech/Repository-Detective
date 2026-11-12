package httpbody

import "io"

// Close closes an HTTP response body and treats a close error as handled.
// Callers use it in place of defer resp.Body.Close() so the error is not ignored.
// A failed close is non-fatal: the response has already been read or discarded.
func Close(body io.Closer) {
	if body == nil {
		return
	}
	_ = body.Close()
}
