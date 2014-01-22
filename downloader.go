package downloader

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"
)

const (
	USER_AGENT = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_8_0) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/31.0.1650.57 Safari/537.36"
)

var TIME_OUT = 5 * time.Second

type Downloader interface {
	Download(url string) (string, error)
}

type HTTPGetDownloader struct {
	cleaner *HTMLCleaner
	client  *http.Client
}

func dialTimeout(network, addr string) (net.Conn, error) {
	return net.DialTimeout(network, addr, TIME_OUT)
}

func NewHTTPGetDownloader() *HTTPGetDownloader {
	ret := HTTPGetDownloader{}
	ret.cleaner = NewHTMLCleaner()
	ret.client = &http.Client{
		Transport: &http.Transport{
			Dial: dialTimeout,
		},
	}
	return &ret
}

func (self *HTTPGetDownloader) Download(url string) (string, error) {
	req, err := http.NewRequest("GET", url, nil)
	req.Header.Set("User-Agent", USER_AGENT)
	if err != nil {
		return "", err
	}
	resp, err := self.client.Do(req)
	if err != nil {
		return "", err
	} else {
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

func (self *WebPage) ToString() string {
	return strconv.FormatInt(self.DownloadedAt, 10) + "\t" + self.Link + "\t" + self.Html
}

type DownloadHandler struct {
	LinksChannel          chan string
	Downloader            *HTTPGetDownloader
	signals               chan os.Signal
	cache                 []*WebPage
	ExtractedLinksChannel chan string
}

func (self *DownloadHandler) FlushCache2Disk() {
	f, err := os.Create("./pages/" + strconv.FormatInt(time.Now().UnixNano(), 10) + ".tsv")
	if err != nil {
		return
	}
	defer f.Close()
	for _, page := range self.cache {
		f.WriteString(page.ToString() + "\n")
	}
	self.cache = []*WebPage{}
}

func (self *DownloadHandler) Download() {
	for link := range self.LinksChannel {
		fmt.Println(link)
		html, err := self.Downloader.Download(link)
		if err != nil {
			fmt.Println(err)
		} else {
			elinks := ExtractLinks([]byte(html), link)
			for _, elink := range elinks {
				self.ExtractedLinksChannel <- elink
			}
			self.cache = append(self.cache, &(WebPage{Link: link, Html: html, DownloadedAt: time.Now().Unix()}))
		}

		if len(self.cache) > 1000 {
			self.FlushCache2Disk()
		}
	}
}

func (self *DownloadHandler) ProcExtractedLinks() {
	tm := time.Now().Unix()
	lm := make(map[string]bool)
	for link := range self.ExtractedLinksChannel {
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
				http.PostForm(ConfigInstance().RedirectorHost, post)
			}
			tm = time.Now().Unix()
			lm = make(map[string]bool)
		}
	}
}

func NewDownloadHanler() *DownloadHandler {
	ret := DownloadHandler{}
	ret.LinksChannel = make(chan string, 10000)
	ret.ExtractedLinksChannel = make(chan string, 10000)
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
			fmt.Println(r)
		}
	}()

	links := req.PostFormValue("links")
	if len(links) > 0 {
		pb := PostBody{}
		json.Unmarshal([]byte(links), &pb)

		for _, link := range pb.Links {
			self.LinksChannel <- link
		}
	}

	ret := Response{
		PostChannelLength:      len(self.LinksChannel),
		ExtractedChannelLength: len(self.ExtractedLinksChannel),
		CacheSize:              len(self.cache),
	}
	output, _ := json.Marshal(&ret)
	fmt.Fprint(w, string(output))
}
