package ioutil

import (
	"io"
	"log/slog"
)

func CloseLogError(c io.Closer) {
	if err := c.Close(); err != nil {
		slog.Warn("close fail", "err", err)
	}
}
