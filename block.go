package downloader

import (
	"strings"
)

func IsBlock(html string) bool {
	if strings.Contains(html, "您的访问出错了") {
		return true
	}
	return false
}