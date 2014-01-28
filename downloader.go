package downloader

import (
	"crawler/downloader/graphite"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"math/rand"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"
)

const (
	USER_AGENT            = "Mozilla/5.0 AppleWebKit/537.36 (KHTML, like Gecko) Chrome/31"
	DOWNLOADER_QUEUE_SIZE = 512
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

	return &ret
}

func (self *HTTPGetDownloader) Download(url string) (string, error) {
	if !IsValidLink(url) {
		return "", nil
	}
	log.Println("download : ", url)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil || req == nil || req.Header == nil {
		return "", err
	}
	req.Header.Set("User-Agent", USER_AGENT)
	resp, err := self.client.Do(req)

	if err != nil || resp == nil || resp.Body == nil {
		return "", err
	} else {
		defer resp.Body.Close()
		if !strings.Contains(resp.Header.Get("Content-Type"), "text/html") {
			return "", errors.New("non html page")
		}

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
	ExtractedLinksChannel chan string
	PageChannel           chan WebPage
	urlFilter             *URLFilter
	writer                *os.File
	currentPath           string
	flushFileSize         int
}

func (self *DownloadHandler) WritePage(page WebPage) {
	if !IsUTF8(page.Link) {
		return
	}
	if !IsUTF8(page.Html) {
		return
	}
	if !strings.Contains(page.Html, ExtractMainDomain(page.Link)) {
		log.Println("html does not have domain", page.Link)
		return
	}
	self.metricSender.Inc("crawler.downloader.save_page_count", 1, 1.0)
	self.writer.WriteString(strconv.FormatInt(page.DownloadedAt, 10))
	self.writer.WriteString("\t")
	self.writer.WriteString(page.Link)
	self.writer.WriteString("\t")
	self.writer.WriteString(page.Html)
	self.writer.WriteString("\n")
}

func (self *DownloadHandler) FlushPages() {
	for page := range self.PageChannel {
		self.WritePage(page)
		self.flushFileSize += 1

		if self.flushFileSize%ConfigInstance().WritePageFreq == 0 {
			defer self.writer.Close()
			os.Rename("./tmp/"+self.currentPath, "./pages/"+self.currentPath)
			self.currentPath = strconv.FormatInt(time.Now().UnixNano(), 10) + ".tsv"
			var err error
			self.writer, err = os.Create("./tmp/" + self.currentPath)
			if err != nil {
				log.Println(err)
				os.Exit(0)
			}
			self.flushFileSize = 0
		}
	}
}

func (self *DownloadHandler) Download() {
	self.flushFileSize = 0
	rand.Seed(time.Now().UnixNano())
	for link0 := range self.LinksChannel {
		go func() {
			link := link0
			log.Println("begin : ", link)
			self.metricSender.Inc("crawler.downloader.tryto_download_count", 1, 1.0)
			html, err := self.Downloader.Download(link)
			if err != nil {
				log.Println(err)
				return
			}
			if len(html) < 100 {
				return
			}
			page := WebPage{Link: link, Html: html, DownloadedAt: time.Now().Unix()}
			if len(self.PageChannel) < DOWNLOADER_QUEUE_SIZE {
				self.PageChannel <- page
			}

			elinks := ExtractLinks([]byte(html), link)
			log.Println("extract links : ", len(elinks))
			for _, elink := range elinks {
				nlink := NormalizeLink(elink)
				if IsValidLink(nlink) && len(self.ExtractedLinksChannel) < DOWNLOADER_QUEUE_SIZE && self.Match(nlink) == 2 {
					self.ExtractedLinksChannel <- nlink
				}
			}
			for _, elink := range elinks {
				nlink := NormalizeLink(elink)
				if IsValidLink(nlink) && len(self.ExtractedLinksChannel) < DOWNLOADER_QUEUE_SIZE && self.Match(nlink) == 1 && rand.Float64() < 0.3 {
					self.ExtractedLinksChannel <- nlink
				}
			}

		}()

	}
}

func (self *DownloadHandler) Match(link string) int {
	return self.urlFilter.Match(link)
}

func (self *DownloadHandler) ProcExtractedLinks() {
	procn := 0
	tm := time.Now().Unix()
	lm := make(map[string]bool)
	for link := range self.ExtractedLinksChannel {
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
				req := make(map[string]string)
				req["links"] = string(jsonBlob)
				PostHTTPRequest(ConfigInstance().RedirectorHost, req)
			}
			tm = time.Now().Unix()
			lm = make(map[string]bool)
		}
		procn += 1
	}
}

func NewDownloadHanler() *DownloadHandler {
	ret := DownloadHandler{}
	ret.urlFilter = NewURLFilter()
	var err error
	ret.currentPath = strconv.FormatInt(time.Now().UnixNano(), 10) + ".tsv"
	ret.writer, err = os.Create("./tmp/" + ret.currentPath)
	if err != nil {
		log.Println(err)
		os.Exit(0)
	}
	ret.metricSender, _ = graphite.New(ConfigInstance().GraphiteHost, "")
	ret.LinksChannel = make(chan string, DOWNLOADER_QUEUE_SIZE)
	ret.PageChannel = make(chan WebPage, DOWNLOADER_QUEUE_SIZE)
	ret.ExtractedLinksChannel = make(chan string, DOWNLOADER_QUEUE_SIZE)
	ret.Downloader = NewHTTPGetDownloader()
	ret.signals = make(chan os.Signal, 1)
	signal.Notify(ret.signals, syscall.SIGINT)
	go func() {
		<-ret.signals
		defer ret.writer.Close()
		os.Rename("./tmp/"+ret.currentPath, "./pages/"+ret.currentPath)
		os.Exit(0)
	}()
	go ret.Download()
	go ret.ProcExtractedLinks()
	go ret.FlushPages()
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
	}
	self.metricSender.Gauge("crawler.downloader."+GetHostName()+"."+Port+".postchannelsize", int64(ret.PostChannelLength), 1.0)
	self.metricSender.Gauge("crawler.downloader."+GetHostName()+"."+Port+".extractchannelsize", int64(ret.ExtractedChannelLength), 1.0)
	self.metricSender.Gauge("crawler.downloader."+GetHostName()+"."+Port+".cachesize", int64(self.flushFileSize), 1.0)
	self.metricSender.Gauge("crawler.downloader."+GetHostName()+"."+Port+".pagechannelsize", int64(len(self.PageChannel)), 1.0)
	output, _ := json.Marshal(&ret)
	fmt.Fprint(w, string(output))
}
