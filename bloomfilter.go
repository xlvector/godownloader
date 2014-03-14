package downloader

import (
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"
)

type BloomFilter struct {
	h        []uint16
	size     int32
	saveChan chan int
}

func Hash(buf string) int32 {
	var seed int32
	var h int32

	seed = 131
	h = 0

	for _, ch := range buf {
		h = h*seed + int32(ch)
	}

	if h < 0 {
		h *= -1
	}
	return h
}

func GetDayTimeStamp() uint16 {
	beginTime := time.Date(2014, 1, 1, 0, 0, 0, 0, time.UTC).Unix()
	nowTime := time.Now().Unix()
	return (uint16)((nowTime - beginTime) / (24 * 3600))
}

func NewBloomFilter() *BloomFilter {
	bf := BloomFilter{}
	bf.size = 500000000
	bf.h = make([]uint16, bf.size)
	for i := int32(0); i < bf.size; i++ {
		bf.h[i] = uint16(0)
	}
	return &bf
}

func (self *BloomFilter) Add(buf string) {
	ha := Hash(buf)
	self.h[ha%self.size] = GetDayTimeStamp()
}

func (self *BloomFilter) Contains(buf string) bool {
	ha := Hash(buf)
	currentDay := GetDayTimeStamp()
	if (currentDay - self.h[ha%self.size]) < 5 {
		return true
	} else {
		return false
	}
}

type BloomFilterHandler struct {
	filter *BloomFilter
}

func NewBloomFilterHandler() *BloomFilterHandler {
	ret := BloomFilterHandler{}
	ret.filter = NewBloomFilter()
	return &ret
}

func (self *BloomFilterHandler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	defer func() {
		if r := recover(); r != nil {
			log.Println(r)
		}
	}()

	link := req.PostFormValue("link")
	method := req.PostFormValue("method")

	if method == "get" {
		if self.filter.Contains(link) {
			fmt.Fprintf(w, "true")
		} else {
			fmt.Fprintf(w, "false")
		}
	} else if method == "set" {
		self.filter.Add(link)
		fmt.Fprintf(w, "")
	}
}

func CheckBloomFilter(link string) bool {
	req := make(map[string]string)
	req["method"] = "get"
	req["link"] = link
	output := strings.Trim(PostHTTPRequest(ConfigInstance().FilterHost, req), "\n")
	if output == "true" {
		return true
	} else {
		return false
	}
}

func SetBloomFilter(link string) {
	req := make(map[string]string)
	req["method"] = "set"
	req["link"] = link
	PostHTTPRequest(ConfigInstance().FilterHost, req)
}
