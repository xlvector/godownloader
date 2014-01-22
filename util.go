package downloader

import (
	"os"
	"strings"
)

func GetHostName() string {
	hostname, err := os.Hostname()
	if err != nil {
		return "unknown"
	}
	return strings.Replace(strings.ToLower(hostname), ".", "_", -1)
}
