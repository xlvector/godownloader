package downloader

import (
	"net"
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
	ret = append(ret, "124.67.215.9:80")
	ret = append(ret, "85.114.141.191:80")
	ret = append(ret, "183.224.1.30:80")
	ret = append(ret, "61.147.82.87:8000")
	ret = append(ret, "61.174.9.96:8080")
	ret = append(ret, "58.20.223.230:3128")
	ret = append(ret, "114.80.136.112:7780")
	ret = append(ret, "119.188.46.42:8080")
	ret = append(ret, "58.20.127.106:3128")
	return ret
}
