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

func extractUrlParams(link string) map[string]string {
	tks := strings.Split(link, "?")
	if len(tks) != 2 {
		return nil
	}
	params := strings.Split(tks[1], "&")
	ret := make(map[string]string)
	for _, param := range params {
		kv := strings.Split(param, "=")
		if len(kv) == 2 {
			ret[kv[0]] = kv[1]
		}
	}
	return ret
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

func IsChinesePage(html string) bool {
	if strings.Contains(html, "的") || strings.Contains(html, "了") || strings.Contains(html, "中") || strings.Contains(html, "页") {
		return true
	}
	if strings.Contains(html, "地") || strings.Contains(html, "是") || strings.Contains(html, "有") || strings.Contains(html, "一") {
		return true
	}
	if strings.Contains(html, "个") || strings.Contains(html, "名") || strings.Contains(html, "欢") || strings.Contains(html, "错") {
		return true
	}
	if strings.Contains(html, "年") || strings.Contains(html, "月") || strings.Contains(html, "日") {
		return true
	}

	return false
}
