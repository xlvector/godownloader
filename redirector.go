package downloader

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"time"
)

const (
	REDIRECT_CHAN_NUM = 512
	PAGE_PER_MINUTE   = 5
)

type RedirectorHandler struct {
	processedLinks *BloomFilter
	linksChannel   []chan string
	patterns       []*regexp.Regexp
}

func (self *RedirectorHandler) Match(link string) bool {
	for _, pt := range self.patterns {
		if pt.FindString(link) == link {
			return true
		}
	}
	return false
}

func (self *RedirectorHandler) Redirect(ci int) {
	for link := range self.linksChannel[ci] {
		if !self.Match(link) {
			continue
		}
		if self.processedLinks.Contains(link) {
			continue
		}
		self.processedLinks.Add(link)

		fmt.Println(link)

		pb := PostBody{}
		pb.Links = []string{link}
		jsonBlob, err := json.Marshal(&pb)
		if err == nil {
			post := url.Values{}
			post.Set("links", string(jsonBlob))

			http.PostForm(ConfigInstance().DownloaderHost, post)
		}
		time.Sleep(60 * time.Second / PAGE_PER_MINUTE)
	}
}

func NewRedirectorHandler() *RedirectorHandler {
	ret := RedirectorHandler{}
	ret.linksChannel = []chan string{}
	for i := 0; i < REDIRECT_CHAN_NUM; i++ {
		ret.linksChannel = append(ret.linksChannel, make(chan string, 10000))
	}
	ret.processedLinks = NewBloomFilter()
	for _, pt := range ConfigInstance().SitePatterns {
		re := regexp.MustCompile(pt)
		ret.patterns = append(ret.patterns, re)
	}
	for i := 0; i < REDIRECT_CHAN_NUM; i++ {
		go ret.Redirect(i)
	}
	return &ret
}

func (self *RedirectorHandler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	defer func() {
		if r := recover(); r != nil {
			fmt.Println(r)
		}
	}()

	links := req.PostFormValue("links")
	if len(links) > 0 {
		pb := PostBody{}
		json.Unmarshal([]byte(links), &pb)

		for _, link := range pb.Links {
			ci := Hash(ExtractDomain(link)) % REDIRECT_CHAN_NUM
			self.linksChannel[ci] <- link
		}
	}

	linkChannelTotalSize := 0
	for _, cn := range self.linksChannel {
		linkChannelTotalSize += len(cn)
	}
	ret := Response{
		PostChannelLength: linkChannelTotalSize,
	}
	output, _ := json.Marshal(&ret)
	fmt.Fprint(w, string(output))
}
