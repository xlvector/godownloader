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

func IsUTF8(buf string) bool {
	i := 0
	for {
		if i >= len(buf) {
			break
		}
		if uint8(buf[i]) > 127 {
			if i+2 >= len(buf) {
				return false
			}
			if uint8(buf[i+1]) <= 127 || uint8(buf[i+2]) <= 127 {
				return false
			}
			i += 3
		} else {
			i += 1
		}
	}
	return true
}
