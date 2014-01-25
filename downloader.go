package downloader

import (
	"crawler/downloader/graphite"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"time"
)

const (
	USER_AGENT            = "Mozilla/5.0 AppleWebKit/537.36 (KHTML, like Gecko) Chrome/31"
	DOWNLOADER_QUEUE_SIZE = 1000
)

type Downloader interface {
	Download(url string) (string, error)
}

type HTTPGetDownloader struct {
	cleaner *HTMLCleaner
	client  *http.Client
}

func dialTimeout(network, addr string) (net.Conn, error) {
	timeout := time.Duration(ConfigInstance().DownloadTimeout) * time.Second
	deadline := time.Now().Add(timeout)
	c, err := net.DialTimeout(network, addr, timeout)
	if err != nil {
		return nil, err
	}
	c.SetDeadline(deadline)
	return c, nil
}

func NewHTTPGetDownloader() *HTTPGetDownloader {
	ret := HTTPGetDownloader{}
	ret.cleaner = NewHTMLCleaner()
	ret.client = &http.Client{
		Transport: &http.Transport{
			Dial:                  dialTimeout,
			DisableKeepAlives:     true,
			ResponseHeaderTimeout: time.Duration(ConfigInstance().DownloadTimeout) * time.Second,
		},
	}
	/*
		proxyList := GetProxyList()
		if len(proxyList) == 0 {
			return &ret
		}
		for i := 0; i < 3; i++ {
			k := (int)(time.Now().UnixNano() % int64(len(proxyList)))
			fmt.Println(proxyList[k])
			if CheckProxy(proxyList[k]) {
				proxyUrl, err := url.Parse(proxyList[k])
				if err != nil {
					continue
				}
				ret.client = &http.Client{
					Transport: &http.Transport{
						Dial:                  dialTimeout,
						DisableKeepAlives:     true,
						ResponseHeaderTimeout: time.Duration(ConfigInstance().DownloadTimeout) * time.Second,
						Proxy: http.ProxyURL(proxyUrl),
					},
				}
				fmt.Println("Use proxy ", proxyList[k])
				break
			}
		}
	*/
	return &ret
}

func (self *HTTPGetDownloader) Download(url string) (string, error) {
	if !IsValidLink(url) {
		return "", nil
	}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil || req == nil || req.Header == nil {
		return "", err
	}
	req.Header.Set("User-Agent", USER_AGENT)
	resp, err := self.client.Do(req)

	if err != nil {
		return "", err
	} else {
		log.Println(url)
		if !strings.Contains(resp.Header.Get("Content-Type"), "text/html") {
			return "", errors.New("non html page")
		}
		defer resp.Body.Close()
		html, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return "", err
		} else {
			utf8Html := self.cleaner.ToUTF8(html)
			if utf8Html == nil {
				return "", errors.New("conver to utf8 error")
			}
			cleanHtml := self.cleaner.CleanHTML(utf8Html)
			return string(cleanHtml), nil
		}
	}
}

type PostBody struct {
	Links []string `json:"links"`
}

type Response struct {
	PostChannelLength      int `json:"post_chan_length"`
	ExtractedChannelLength int `json:"extract_chan_length"`
	CacheSize              int `json:"cache_size"`
}

type WebPage struct {
	Link         string
	Html         string
	DownloadedAt int64
}

type DownloadHandler struct {
	metricSender          *graphite.Client
	LinksChannel          chan string
	Downloader            *HTTPGetDownloader
	signals               chan os.Signal
	cache                 []*WebPage
	ExtractedLinksChannel chan string
	patterns              []*regexp.Regexp
}

func (self *DownloadHandler) FlushCache2Disk() {
	path := strconv.FormatInt(time.Now().UnixNano(), 10) + ".tsv"
	f, err := os.Create("./tmp/" + path)
	if err != nil {
		return
	}
	for _, page := range self.cache {
		if !IsUTF8(page.Link) {
			continue
		}
		if !IsUTF8(page.Html) {
			continue
		}
		f.WriteString(strconv.FormatInt(page.DownloadedAt, 10))
		f.WriteString("\t")
		f.WriteString(page.Link)
		f.WriteString("\t")
		f.WriteString(page.Html)
		f.WriteString("\n")
		page = nil
	}
	f.Close()
	os.Rename("./tmp/"+path, "./pages/"+path)
	self.cache = []*WebPage{}
	runtime.GC()
}

func (self *DownloadHandler) Download() {
	for link := range self.LinksChannel {
		html, err := self.Downloader.Download(link)
		if err != nil {
			log.Println(err)
			continue
		}
		if len(html) < 100 {
			continue
		}

		self.cache = append(self.cache, &(WebPage{Link: link, Html: html, DownloadedAt: time.Now().Unix()}))
		elinks := ExtractLinks([]byte(html), link)
		log.Println("extract links : ", len(elinks))
		for _, elink := range elinks {
			nlink := NormalizeLink(elink)
			if IsValidLink(nlink) && len(self.ExtractedLinksChannel) < DOWNLOADER_QUEUE_SIZE {
				self.ExtractedLinksChannel <- nlink
			}
		}

		if len(self.cache) > 200 {
			self.FlushCache2Disk()
		}
	}
}

func (self *DownloadHandler) Match(link string) bool {
	for _, pt := range self.patterns {
		if pt.FindString(link) == link {
			return true
		}
	}
	return false
}

func (self *DownloadHandler) ProcExtractedLinks() {
	procn := 0
	tm := time.Now().Unix()
	lm := make(map[string]bool)
	for link := range self.ExtractedLinksChannel {
		if !self.Match(link) {
			continue
		}
		lm[link] = true
		tm1 := time.Now().Unix()

		if tm1-tm > 60 || len(lm) > 100 || procn < 10 {
			log.Println("send links : ", len(lm))
			pb := PostBody{}
			pb.Links = []string{}
			for lk, _ := range lm {
				pb.Links = append(pb.Links, lk)
			}
			jsonBlob, err := json.Marshal(&pb)
			if err == nil {
				post := url.Values{}
				post.Set("links", string(jsonBlob))
				resp, err := http.PostForm(ConfigInstance().RedirectorHost, post)

				if err != nil {
					log.Println(err)
				}
				if resp != nil && resp.Body != nil {
					ioutil.ReadAll(resp.Body)
					resp.Body.Close()
				}
			}
			tm = time.Now().Unix()
			lm = make(map[string]bool)
		}
		procn += 1
	}
}

func NewDownloadHanler() *DownloadHandler {
	ret := DownloadHandler{}
	for _, pt := range ConfigInstance().SitePatterns {
		re := regexp.MustCompile(pt)
		ret.patterns = append(ret.patterns, re)
	}
	ret.metricSender, _ = graphite.New(ConfigInstance().GraphiteHost, "")
	ret.LinksChannel = make(chan string, DOWNLOADER_QUEUE_SIZE)
	ret.ExtractedLinksChannel = make(chan string, DOWNLOADER_QUEUE_SIZE)
	ret.Downloader = NewHTTPGetDownloader()
	ret.signals = make(chan os.Signal, 1)
	signal.Notify(ret.signals, syscall.SIGINT)
	go func() {
		<-ret.signals
		ret.FlushCache2Disk()
		os.Exit(0)
	}()
	ret.cache = []*WebPage{}
	go ret.Download()
	go ret.ProcExtractedLinks()
	return &ret
}

func (self *DownloadHandler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
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
			if len(self.LinksChannel) < DOWNLOADER_QUEUE_SIZE {
				self.LinksChannel <- link
			}
		}
	}

	ret := Response{
		PostChannelLength:      len(self.LinksChannel),
		ExtractedChannelLength: len(self.ExtractedLinksChannel),
		CacheSize:              len(self.cache),
	}
	self.metricSender.Gauge("crawler.downloader."+GetHostName()+"."+Port+".postchannelsize", int64(ret.PostChannelLength), 1.0)
	self.metricSender.Gauge("crawler.downloader."+GetHostName()+"."+Port+".extractchannelsize", int64(ret.ExtractedChannelLength), 1.0)
	self.metricSender.Gauge("crawler.downloader."+GetHostName()+"."+Port+".cachesize", int64(ret.CacheSize), 1.0)
	output, _ := json.Marshal(&ret)
	fmt.Fprint(w, string(output))
}
