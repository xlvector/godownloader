package downloader

import (
	"bufio"
	"net"
	"os"
	"strings"
	"time"
)

func CheckProxy(proxy string) bool {
	_, err := net.DialTimeout("tcp", proxy, time.Millisecond*5000)
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
		line = strings.Trim(line, "\n")
		if !strings.Contains(line, "http://") {
			line = "http://" + line
		}
		ret = append(ret, line)
	}
	return ret
}

func GetRealtimeDownloaderList() []string {
	ret := []string{}
	f, err := os.Open("realtime_downloader.list")
	if err != nil {
		return ret
	}
	r := bufio.NewReader(f)
	for {
		line, err := r.ReadString('\n')
		if err != nil {
			break
		}
		line = strings.Trim(line, "\n")
		ret = append(ret, line)
	}
	return ret
}
