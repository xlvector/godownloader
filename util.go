package downloader

import (
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"net/url"
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

func LoopUpIp(host string) string {
	ips, err := net.LookupIP(host)
	if err != nil {
		return ""
	}
	for _, ip := range ips {
		return string(ip)
	}
	return ""
}

func PostHTTPRequest(host string, data map[string]string) string {
	post := url.Values{}
	for key, value := range data {
		post.Set(key, value)
	}
	resp, err := http.PostForm(host, post)

	if err != nil {
		log.Println(err)
	}
	if resp != nil && resp.Body != nil {
		defer resp.Body.Close()
		output, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			log.Println(err)
		}
		return string(output)
	}
	return ""
}
