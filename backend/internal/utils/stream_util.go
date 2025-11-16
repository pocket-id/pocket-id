package utils

import (
	"errors"
	"io"
)

var ErrSizeExceeded = errors.New("stream size exceeded")

// LimitReader is like io.LimitReader but throws an error if the stream exceeds the max size
// io.LimitReader instead just returns io.EOF
// Adapted from https://github.com/golang/go/issues/51115#issuecomment-1079761212
type LimitReader struct {
	io.ReadCloser
	N int64
}

func NewLimitReader(r io.ReadCloser, limit int64) *LimitReader {
	return &LimitReader{r, limit}
}

func (r *LimitReader) Read(p []byte) (n int, err error) {
	if r.N <= 0 {
		return 0, ErrSizeExceeded
	}

	if int64(len(p)) > r.N {
		p = p[0:r.N]
	}

	n, err = r.ReadCloser.Read(p)
	r.N -= int64(n)
	return
}
