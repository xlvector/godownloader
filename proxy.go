package downloader

import (
	"bufio"
	"net"
	"os"
	"strings"
	"time"
)

func CheckProxy(proxy string) bool {
	_, err := net.DialTimeout("tcp", proxy, time.Millisecond*100)
	if err != nil {
		return false
	}
	return true
}

func GetProxyList() []string {
	ret := []string{}
	f, err := os.Open("proxy.list")
	if err != nil {
		return ret
	}
	r := bufio.NewReader(f)
	for {
		line, err := r.ReadString('\n')
		if err != nil {
			break
		}
		ret = append(ret, strings.Trim(line, "\n"))
	}
	return ret
}
