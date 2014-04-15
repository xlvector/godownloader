package downloader

import (
	"fmt"
	"os"
)

LogWriter, _ := os.Create("access.log")

func Logging(timestamp int64, host, action, link string) {
	fmt.Fprintf(LogWriter, "%d %s %s %s\n", timestamp, host, action, link)
}