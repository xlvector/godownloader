package downloader

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"
)

type Downloader interface {
	Download(url string) (string, error)
}

type HTTPGetDownloader struct {
	cleaner *HTMLCleaner
}

func NewHTTPGetDownloader() *HTTPGetDownloader {
	ret := HTTPGetDownloader{}
	ret.cleaner = NewHTMLCleaner()
	return &ret
}

func (self *HTTPGetDownloader) Download(url string) (string, error) {
	resp, err := http.Get(url)
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
	ChannelLength int `json:"channel_length"`
	CacheSize     int `json:"cache_size"`
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
	LinksChannel chan string
	Downloader   *HTTPGetDownloader
	signals      chan os.Signal
	cache        []*WebPage
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
			self.cache = append(self.cache, &(WebPage{Link: link, Html: html, DownloadedAt: time.Now().Unix()}))
		}

		if len(self.cache) > 1000 {
			self.FlushCache2Disk()
		}
	}
}

func NewDownloadHanler() *DownloadHandler {
	ret := DownloadHandler{}
	ret.LinksChannel = make(chan string, 10000)
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
		ChannelLength: len(self.LinksChannel),
		CacheSize:     len(self.cache),
	}
	output, _ := json.Marshal(&ret)
	fmt.Fprint(w, string(output))
}
