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
	"net/url"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"
	"unicode/utf8"
)

const (
	USER_AGENT            = "Mozilla/5.0 (Windows; U; Windows NT 5.1; zh-CN; rv:1.8.1.14) Gecko/20080404 (FoxPlus) Firefox/2.0.0.14"
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

func NewHTTPGetProxyDownloader(proxy string) *HTTPGetDownloader {
	ret := HTTPGetDownloader{}
	ret.cleaner = NewHTMLCleaner()
	proxyUrl, err := url.Parse(proxy)
	if err != nil {
		return nil
	}
	ret.client = &http.Client{
		Transport: &http.Transport{
			Dial:                  dialTimeout,
			DisableKeepAlives:     true,
			ResponseHeaderTimeout: time.Duration(ConfigInstance().DownloadTimeout) * time.Second,
			Proxy: http.ProxyURL(proxyUrl),
		},
	}
	return &ret
}

func NewDefaultHTTPGetProxyDownloader(proxy string) *HTTPGetDownloader {
	ret := HTTPGetDownloader{}
	ret.cleaner = nil
	proxyUrl, err := url.Parse(proxy)
	if err != nil {
		return nil
	}
	ret.client = &http.Client{
		Transport: &http.Transport{
			DisableKeepAlives: true,
			Proxy:             http.ProxyURL(proxyUrl),
		},
	}
	return &ret
}

func (self *HTTPGetDownloader) Download(url string) (string, string, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil || req == nil || req.Header == nil {
		return "", "", err
	}
	req.Header.Set("User-Agent", USER_AGENT)
	resp, err := self.client.Do(req)

	respInfo := ""
	if err != nil || resp == nil || resp.Body == nil {
		return "", "", err
	} else {
		respInfo += "<real_url>" + resp.Request.URL.String() + "</real_url>"
		respInfo += "<content_type>" + resp.Header.Get("Content-Type") + "</content_type>"
		defer resp.Body.Close()
		if !strings.Contains(resp.Header.Get("Content-Type"), "text/") && !strings.Contains(resp.Header.Get("Content-Type"), "json") {
			return "", "", errors.New("non html page")
		}
		cleanRespInfo := string(self.cleaner.CleanHTML([]byte(respInfo)))
		html, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return "", "", err
		} else {
			if self.cleaner != nil {
				utf8Html := self.cleaner.ToUTF8(html)
				if utf8Html == nil {
					return "", "", errors.New("conver to utf8 error")
				}
				cleanHtml := self.cleaner.CleanHTML(utf8Html)
				return string(cleanHtml), cleanRespInfo, nil
			} else {
				return string(html), cleanRespInfo, nil
			}
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
	RespInfo     string
	DownloadedAt int64
}

type WebSiteStat struct {
	linkRecvCount     map[string]int
	pageDownloadCount map[string]int
	pageWriteCount    map[string]int
	ruleMatcher       *RuleMatcher
}

type DownloadHandler struct {
	ticker *time.Ticker
	metricSender                   *graphite.Client
	LinksChannel                   chan string
	Downloader                     *HTTPGetDownloader
	ProxyDownloader                []*HTTPGetDownloader
	signals                        chan os.Signal
	ExtractedLinksChannel          chan string
	PageChannel                    chan WebPage
	urlFilter                      *URLFilter
	writer                         *os.File
	currentPath                    string
	flushFileSize                  int
	processedPageCount             int
	totalDownloadedPageCount       int
	proxyDownloadedPageCount       int
	proxyDownloadedPageFailedCount int
	writePageCount                 int
	WebSiteStat
}

func (self *DownloadHandler) WritePage(page WebPage) {

	if !utf8.ValidString(page.Link) {
		log.Println("non utf8 link : ", page.Link)
		return
	}

	if !utf8.ValidString(page.Html) {
		log.Println("non utf8 page : ", page.Link)
		return
	}

	SetBloomFilter(page.Link)

	self.writePageCount += 1
	if self.ruleMatcher.MatchRule(page.Link) == 2 {
		domain := ExtractMainDomain(page.Link)
		domain = strings.Replace(domain, ".", "_", -1)
		self.pageWriteCount[domain] += 1
	}

	self.writer.WriteString(strconv.FormatInt(page.DownloadedAt, 10))
	self.writer.WriteString("\t")
	self.writer.WriteString(page.Link)
	self.writer.WriteString("\t")
	self.writer.WriteString(page.Html)
	self.writer.WriteString("\t")
	self.writer.WriteString(page.RespInfo)
	self.writer.WriteString("\n")
	log.Println(time.Now().Unix(), "downloader", "write", page.Link)
}

func (self *DownloadHandler) FlushPages() {
	for page := range self.PageChannel {
		self.WritePage(page)
		self.flushFileSize += 1

		writePageFreq := ConfigInstance().WritePageFreq
		if writePageFreq > 0 && self.flushFileSize%writePageFreq == 0 {
			self.writer.Close()
			self.currentPath = strconv.FormatInt(time.Now().UnixNano(), 10) + ".tsv"
			var err error
			self.writer, err = os.Create("./pages/" + self.currentPath)
			if err != nil {
				log.Println(err)
				os.Exit(0)
			}
			self.flushFileSize = 0
		}
	}
}

func (self *DownloadHandler) GetProxyDownloader() *HTTPGetDownloader {
	if len(self.ProxyDownloader) == 0 {
		return nil
	}
	return self.ProxyDownloader[rand.Intn(len(self.ProxyDownloader))]
}

func (self *DownloadHandler) UseProxy(link string) bool {
	return true
	/*domain := ExtractMainDomain(link)
	if strings.Contains(domain, "edu.cn") || strings.Contains(domain, "gov.cn") {
		return false
	} else {
		return true
	}*/
}

func (self *DownloadHandler) ProcessLink(link string) {
	if !IsValidLink(link) {
		return
	}
	log.Println(time.Now().Unix(), "downloader", "start", link)
	self.processedPageCount += 1
	html := ""
	resp := ""
	var err error
	downloader := self.GetProxyDownloader()
	if self.UseProxy(link) && downloader != nil {
		html, resp, err = downloader.Download(link)
		if err != nil {
			log.Println("proxy", err)
			self.proxyDownloadedPageFailedCount += 1
			html, resp, err = self.Downloader.Download(link)
		} else {
			self.proxyDownloadedPageCount += 1
		}
	} else {
		html, resp, err = self.Downloader.Download(link)
	}

	if err != nil {
		log.Println(err)
		return
	}
	self.totalDownloadedPageCount += 1

	if self.ruleMatcher.MatchRule(link) == 2 {
		domain := ExtractMainDomain(link)
		domain = strings.Replace(domain, ".", "_", -1)
		self.pageDownloadCount[domain] += 1
	}

	if len(html) < 100 {
		return
	}

	if !IsChinesePage(html) {
		return
	}
	log.Println(time.Now().Unix(), "downloader", "finish", link)
	page := WebPage{Link: link, Html: html, RespInfo: resp, DownloadedAt: time.Now().Unix()}

	if len(self.PageChannel) < DOWNLOADER_QUEUE_SIZE {
		self.PageChannel <- page
	}

	elinks := ExtractLinks([]byte(html), link)
	log.Println("extract links : ", len(elinks))
	for _, elink := range elinks {
		nlink := NormalizeLink(elink)
		linkPriority := self.Match(nlink)
		if linkPriority <= 0 {
			continue
		}
		if IsValidLink(nlink) && len(self.ExtractedLinksChannel) < DOWNLOADER_QUEUE_SIZE {
			self.ExtractedLinksChannel <- nlink
		}
	}
}

func (self *DownloadHandler) Download() {
	self.flushFileSize = 0
	rand.Seed(time.Now().UnixNano())
	for link0 := range self.LinksChannel {
		go self.ProcessLink(link0)
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
	ret.writer, err = os.Create("./pages/" + ret.currentPath)
	defer ret.writer.Close()

	if err != nil {
		log.Println(err)
		os.Exit(0)
	}
	ret.metricSender, _ = graphite.New(ConfigInstance().GraphiteHost, "")
	ret.LinksChannel = make(chan string, DOWNLOADER_QUEUE_SIZE)
	ret.PageChannel = make(chan WebPage, DOWNLOADER_QUEUE_SIZE)
	ret.ExtractedLinksChannel = make(chan string, DOWNLOADER_QUEUE_SIZE)
	ret.Downloader = NewHTTPGetDownloader()
	ret.processedPageCount = 0
	ret.totalDownloadedPageCount = 0
	ret.proxyDownloadedPageCount = 0
	ret.writePageCount = 0
	ret.linkRecvCount = make(map[string]int)
	ret.pageDownloadCount = make(map[string]int)
	ret.pageWriteCount = make(map[string]int)
	ret.ruleMatcher = NewRuleMatcher()

	for _, proxy := range GetProxyList() {
		pd := NewHTTPGetProxyDownloader(proxy)
		if pd == nil {
			continue
		}
		ret.ProxyDownloader = append(ret.ProxyDownloader, pd)
	}
	log.Println("proxy downloader count", len(ret.ProxyDownloader))

	ret.ticker = time.NewTicker(time.Second * 60)
	go func() {
		for t := range ret.ticker.C {
			log.Println("refresh rules at", t)
			newRules := GetSitePatterns()
			for rule, pri := range newRules {
				log.Println("add rule", rule, "with priority", pri)
				ret.urlFilter.ruleMatcher.AddRule(rule, pri)
			}
		}
	}()

	ret.signals = make(chan os.Signal, 1)
	signal.Notify(ret.signals, syscall.SIGINT)
	go func() {
		<-ret.signals
		defer ret.writer.Close()
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
			if self.ruleMatcher.MatchRule(link) > 0 {
				domain := ExtractMainDomain(link)
				domain = strings.Replace(domain, ".", "_", -1)
				self.linkRecvCount[domain] += 1
			}

			if len(self.LinksChannel) < DOWNLOADER_QUEUE_SIZE {
				self.LinksChannel <- link
			}
		}
	}

	ret := Response{
		PostChannelLength:      len(self.LinksChannel),
		ExtractedChannelLength: len(self.ExtractedLinksChannel),
	}
	if rand.Float64() < 0.1 {

		self.metricSender.Gauge("crawler.downloader."+GetHostName()+"."+Port+".postchannelsize", int64(ret.PostChannelLength), 1.0)
		self.metricSender.Gauge("crawler.downloader."+GetHostName()+"."+Port+".extractchannelsize", int64(ret.ExtractedChannelLength), 1.0)
		self.metricSender.Gauge("crawler.downloader."+GetHostName()+"."+Port+".cachesize", int64(self.flushFileSize), 1.0)
		self.metricSender.Gauge("crawler.downloader."+GetHostName()+"."+Port+".pagechannelsize", int64(len(self.PageChannel)), 1.0)
		self.metricSender.Gauge("crawler.downloader."+GetHostName()+"."+Port+".totalDownloadedPageCount", int64(self.totalDownloadedPageCount), 1.0)
		self.metricSender.Gauge("crawler.downloader."+GetHostName()+"."+Port+".processedPageCount", int64(self.processedPageCount), 1.0)
		self.metricSender.Gauge("crawler.downloader."+GetHostName()+"."+Port+".proxyDownloadedPageCount", int64(self.proxyDownloadedPageCount), 1.0)
		self.metricSender.Gauge("crawler.downloader."+GetHostName()+"."+Port+".proxyDownloadedPageFailedCount", int64(self.proxyDownloadedPageFailedCount), 1.0)
		self.metricSender.Gauge("crawler.downloader."+GetHostName()+"."+Port+".writePageCount", int64(self.writePageCount), 1.0)
		for domain, downcount := range self.pageDownloadCount {
			metricName := "crawler.downloader." + GetHostName() + "." + Port + ".domainDownloadPageCount." + domain
			self.metricSender.Gauge(metricName, int64(downcount), 1.0)
		}
		for domain, writecount := range self.pageWriteCount {
			metricName := "crawler.downloader." + GetHostName() + "." + Port + ".domainWritePageCount." + domain
			self.metricSender.Gauge(metricName, int64(writecount), 1.0)
		}
		for domain, recvcount := range self.linkRecvCount {
			metricName := "crawler.downloader." + GetHostName() + "." + Port + ".domainLinkRecvCount." + domain
			self.metricSender.Gauge(metricName, int64(recvcount), 1.0)
		}
	}
	output, _ := json.Marshal(&ret)
	fmt.Fprint(w, string(output))
}
