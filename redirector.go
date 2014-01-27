package downloader

import (
	"crawler/downloader/graphite"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"math/rand"
	"net/http"
	"net/url"
	"regexp"
	"time"
)

type RedirectorHandler struct {
	metricSender   *graphite.Client
	processedLinks *BloomFilter
	linksChannel   []chan string
	patterns       []*regexp.Regexp
	dnsCache       map[string]string
}

func (self *RedirectorHandler) Match(link string) bool {
	for _, pt := range self.patterns {
		if pt.FindString(link) == link {
			return true
		}
	}
	return false
}

func (self *RedirectorHandler) GetIP(host string) string {
	ip, ok := self.dnsCache[host]
	if ok {
		return ip
	}
	ip = LoopUpIp(host)
	if ip == "" {
		ip = host
	}
	log.Println("dns lookup", host, ip)
	return ip
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

		log.Println("redirect : ", link)

		pb := PostBody{}
		pb.Links = []string{link}
		jsonBlob, err := json.Marshal(&pb)
		if err == nil {
			post := url.Values{}
			post.Set("links", string(jsonBlob))

			resp, err := http.PostForm(ConfigInstance().DownloaderHost, post)
			if err != nil {
				log.Println(err)
				continue
			}

			if resp != nil && resp.Body != nil {
				ioutil.ReadAll(resp.Body)
				resp.Body.Close()
			}
		}
		time.Sleep(60 * time.Second / time.Duration(ConfigInstance().PagePerMinute))
	}
}

func NewRedirectorHandler() *RedirectorHandler {
	ret := RedirectorHandler{}
	ret.metricSender, _ = graphite.New(ConfigInstance().GraphiteHost, "")
	ret.linksChannel = []chan string{}
	for i := 0; i < ConfigInstance().RedirectChanNum; i++ {
		ret.linksChannel = append(ret.linksChannel, make(chan string, ConfigInstance().RedirectChanSize))
	}
	ret.processedLinks = NewBloomFilter()
	for _, pt := range ConfigInstance().SitePatterns {
		re := regexp.MustCompile(pt)
		ret.patterns = append(ret.patterns, re)
	}
	for i := 0; i < ConfigInstance().RedirectChanNum; i++ {
		go ret.Redirect(i)
	}
	return &ret
}

func (self *RedirectorHandler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	defer func() {
		if r := recover(); r != nil {
			log.Println(r)
		}
	}()

	links := req.PostFormValue("links")
	if len(links) > 0 {
		pb := PostBody{}
		json.Unmarshal([]byte(links), &pb)

		for _, link := range pb.Links {
			if !self.Match(link) {
				continue
			}
			if self.processedLinks.Contains(link) {
				continue
			}
			if rand.Float64() < 0.5 {
				continue
			}
			ci := Hash(ExtractMainDomain(link)) % int32(ConfigInstance().RedirectChanNum)
			if len(self.linksChannel[ci]) < ConfigInstance().RedirectChanSize {
				log.Println("channel ", ci, " recv link : ", link)
				self.linksChannel[ci] <- link
			}
		}
	}
	linkChannelTotalSize := 0

	maxChannelSize := 0

	for _, cn := range self.linksChannel {

		size := len(cn)

		linkChannelTotalSize += size

		if maxChannelSize < size {
			maxChannelSize = size
		}
	}
	self.metricSender.Gauge("crawler.redirector."+GetHostName()+"."+Port+".channelsize", int64(linkChannelTotalSize), 1.0)
	self.metricSender.Gauge("crawler.redirector."+GetHostName()+"."+Port+".maxchannelsize", int64(maxChannelSize), 1.0)
	fmt.Fprint(w, "")
}
