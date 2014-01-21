package downloader

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"time"
)

type RedirectorHandler struct {
	processedLinks *BloomFilter
	linksChannel   chan string
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

func (self *RedirectorHandler) Redirect() {
	lm := make(map[string]bool)
	tm := time.Now().Unix()
	fmt.Println(ConfigInstance().DownloaderHost)
	for link := range self.linksChannel {
		if !self.Match(link) {
			continue
		}
		fmt.Println(link)
		if self.processedLinks.Contains(link) {
			continue
		}
		self.processedLinks.Add(link)
		lm[link] = true
		tm1 := time.Now().Unix()
		if tm1-tm > 60 || len(lm) > 100 {
			pb := PostBody{}
			pb.Links = []string{}
			for lk, _ := range lm {
				pb.Links = append(pb.Links, lk)
			}
			jsonBlob, err := json.Marshal(&pb)
			if err == nil {
				post := url.Values{}
				post.Set("links", string(jsonBlob))

				http.PostForm(ConfigInstance().DownloaderHost, post)
			}
			tm = time.Now().Unix()
			lm = make(map[string]bool)
		}
	}
}

func NewRedirectorHandler() *RedirectorHandler {
	ret := RedirectorHandler{}
	ret.linksChannel = make(chan string, 10000)
	ret.processedLinks = NewBloomFilter()
	for _, pt := range ConfigInstance().SitePatterns {
		re := regexp.MustCompile(pt)
		ret.patterns = append(ret.patterns, re)
	}
	go ret.Redirect()
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
			self.linksChannel <- link
		}
	}

	ret := Response{
		PostChannelLength: len(self.linksChannel),
	}
	output, _ := json.Marshal(&ret)
	fmt.Fprint(w, string(output))
}
