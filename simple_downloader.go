package downloader

import (
	"log"
	"math/rand"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"
	"unicode/utf8"
)

type SimpleDownloadHandler struct {
	ticker *time.Ticker
	LinksChannel                   chan string
	Downloader                     *HTTPGetDownloader
	PageChannel                    chan WebPage
	writer                         *os.File
	currentPath                    string
	signals						chan os.Signal
	flushFileSize	int
}

func (self *SimpleDownloadHandler) WritePage(page WebPage) {

	if !utf8.ValidString(page.Link) {
		return
	}

	if !utf8.ValidString(page.Html) {
		return
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

func (self *SimpleDownloadHandler) FlushPages() {
	self.flushFileSize = 0
	for page := range self.PageChannel {
		self.WritePage(page)
		self.flushFileSize += 1

		writePageFreq := 100
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

func (self *SimpleDownloadHandler) ProcessLink(link string) {
	log.Println(time.Now().Unix(), "downloader", "start", link)
	html := ""
	resp := ""
	var err error	

	html, resp, err = self.Downloader.Download(link)
	if err != nil {
		return
	}

	if len(html) < 100 {
		return
	}

	if !IsChinesePage(html) {
		return
	}
	log.Println(time.Now().Unix(), "downloader", "finish", link)
	page := WebPage{Link: link, Html: html, RespInfo: resp, DownloadedAt: time.Now().Unix()}

	self.PageChannel <- page
}

func (self *SimpleDownloadHandler) Download() {
	rand.Seed(time.Now().UnixNano())
	for link0 := range self.LinksChannel {
		go self.ProcessLink(link0)
	}
}

func NewSimpleDownloadHandler() *SimpleDownloadHandler {
	ret := SimpleDownloadHandler{}
	var err error
	ret.currentPath = strconv.FormatInt(time.Now().UnixNano(), 10) + ".tsv"
	ret.writer, err = os.Create("./pages/" + ret.currentPath)

	if err != nil {
		os.Exit(0)
	}
	ret.LinksChannel = make(chan string, DOWNLOADER_QUEUE_SIZE)
	ret.PageChannel = make(chan WebPage, DOWNLOADER_QUEUE_SIZE)
	ret.Downloader = NewHTTPGetDownloader()

	ret.signals = make(chan os.Signal, 1)
	signal.Notify(ret.signals, syscall.SIGINT)
	go func() {
		<-ret.signals
		defer ret.writer.Close()
		os.Exit(0)
	}()
	go ret.Download()
	go ret.FlushPages()
	return &ret
}

func (self *SimpleDownloadHandler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	defer func() {
		if r := recover(); r != nil {
			log.Println(r)
		}
	}()

	link := req.FormValue("link")
	self.LinksChannel <- link
	fmt.Fprint(w, "got data")
}
