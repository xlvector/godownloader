package downloader

import (
	"bytes"
	_ "code.google.com/p/go.image/bmp"
	_ "code.google.com/p/go.image/tiff"
	"encoding/base64"
	"fmt"
	"image"
	_ "image/color"
	_ "image/gif"
	_ "image/jpeg"
	"image/png"
	"io/ioutil"
	"log"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"
	"unicode/utf8"
	l4g "code.google.com/p/log4go"
)

type PicDownloadHandler struct {
	ticker        *time.Ticker
	LinksChannel  chan string
	Downloader    *BinaryHTTPGetter
	PageChannel   chan WebPage
	writer        *os.File
	currentPath   string
	signals       chan os.Signal
	flushFileSize int
}

func (self *PicDownloadHandler) WritePage(page WebPage) {
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
	l4g.Info(time.Now().Unix(), "downloader", "write", page.Link)
}

func (self *PicDownloadHandler) FlushPages() {
	self.flushFileSize = 0
	for page := range self.PageChannel {
		self.WritePage(page)
		self.flushFileSize += 1

		writePageFreq := 1000
		if writePageFreq > 0 && self.flushFileSize%writePageFreq == 0 {
			self.writer.Close()
			self.currentPath = strconv.FormatInt(time.Now().UnixNano(), 10) + ".tsv"
			var err error
			self.writer, err = os.Create("./images/" + self.currentPath)
			if err != nil {
				log.Println(err)
				os.Exit(0)
			}
			self.flushFileSize = 0
		}
	}
}

func LoadImageFromURL(link string, proxy string) image.Image {
	req, err := http.NewRequest("GET", link, nil)
	if err != nil || req == nil || req.Header == nil {
		return nil
	}
	client := &http.Client{}
	if proxy != "" {
		proxyUrl, _ := url.Parse(proxy)
		client = &http.Client{
			Transport: &http.Transport{
				Dial:                  dialTimeout,
				DisableKeepAlives:     true,
				ResponseHeaderTimeout: 10 * time.Second,
				Proxy: http.ProxyURL(proxyUrl),
			},
		}
	}
	req.Header.Add("User-Agent", "Mozilla/5.0 (Windows NT 6.1; WOW64; rv:12.0) Gecko/20100101 Firefox/12.0")
	resp, err := client.Do(req)

	if err != nil || resp == nil || resp.Body == nil {
		return nil
	} else {
		defer resp.Body.Close()
		img, _, err := image.Decode(resp.Body)
		if err != nil {
			l4g.Warn(err)
			img = nil
		}
		return img
	}
}

func (self *PicDownloadHandler) ProcessLink(link string) {
	l4g.Info(time.Now().Unix(), "downloader", "start", link)
	html := ""
	resp := ""

	memImage := LoadImageFromURL(link, "")
	if memImage != nil {
		buf := new(bytes.Buffer)
		err := png.Encode(buf, memImage)
		
		if err == nil {
			l4g.Info(time.Now().Unix(), "downloader", "finish", link)
			html = base64.StdEncoding.EncodeToString(buf.Bytes())
			page := WebPage{Link: link, Html: html, RespInfo: resp, DownloadedAt: time.Now().Unix()}

			self.PageChannel <- page
		} else {
			l4g.Warn("Unable to load image for "+link+". ", err)
		}
	} else {
		l4g.Warn("Unable to load image for " + link)
	}
	return
}

func (self *PicDownloadHandler) Download() {
	rand.Seed(time.Now().UnixNano())
	for link0 := range self.LinksChannel {
		go self.ProcessLink(link0)
	}
}

func NewPicDownloadHandler() *PicDownloadHandler {
	l4g.LoadConfiguration("log4go.xml")
	
	ret := PicDownloadHandler{}
	var err error
	ret.currentPath = strconv.FormatInt(time.Now().UnixNano(), 10) + ".tsv"
	ret.writer, err = os.Create("./images/" + ret.currentPath)

	if err != nil {
		log.Fatalln(err)
		os.Exit(0)
	}
	ret.LinksChannel = make(chan string, DOWNLOADER_QUEUE_SIZE)
	ret.PageChannel = make(chan WebPage, DOWNLOADER_QUEUE_SIZE)
	ret.Downloader = NewBinaryHTTPGetter()

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

func (self *PicDownloadHandler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	defer func() {
		if r := recover(); r != nil {
			log.Println(r)
		}
	}()

	link := req.FormValue("link")
	repeat := 1
	requestRepeat := req.FormValue("repeat")
	if requestRepeat != "" {
		var err error
		repeat, err = strconv.Atoi(requestRepeat)
		if err != nil {
			log.Println(err)
			repeat = 1
		}
	}
	for i := 0; i < repeat; i++ {
		self.LinksChannel <- link
	}
	fmt.Fprint(w, "got link "+link)
}

type BinaryHTTPGetter struct {
	client *http.Client
}

func NewBinaryHTTPGetter() *BinaryHTTPGetter {
	ret := BinaryHTTPGetter{}
	ret.client = &http.Client{
		Transport: &http.Transport{
			DisableKeepAlives:     true,
			ResponseHeaderTimeout: time.Duration(ConfigInstance().DownloadTimeout) * time.Second,
		},
	}
	return &ret
}

func (self *BinaryHTTPGetter) Download(url string) ([]byte, error) {
	resp, err := self.client.Get(url)
	if err != nil || resp == nil || resp.Body == nil {
		return nil, err
	}
	picBinary, err := ioutil.ReadAll(resp.Body)
	if err != nil || picBinary == nil || len(picBinary) == 0 {
		return nil, err
	}

	return picBinary, nil
}
