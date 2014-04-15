package downloader

import (
	"fmt"
	"io"
)

var LogWriter io.Writer

func Logging(timestamp int64, host, action, link string) {
	fmt.Fprintf(LogWriter, "%d %s %s %s\n", timestamp, host, action, link)
}