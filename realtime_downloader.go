package downloader

import (
	"log"
	"net/http"
	"time"
	"compress/gzip"
	"encoding/base64"
)

type RealtimeDownloadHandler struct {
	Downloader                     *HTTPGetDownloader
}

func (self *RealtimeDownloadHandler) ProcessLink(link string) string {
	log.Println(time.Now().Unix(), "downloader", "start", link)
	html := ""
	var err error	

	html, _, err = self.Downloader.Download(link)
	if err != nil {
		return ""
	}

	if len(html) < 100 {
		return ""
	}

	if !IsChinesePage(html) {
		return ""
	}
	log.Println(time.Now().Unix(), "downloader", "finish", link)
	return html
}


func NewRealtimeDownloadHandler() *RealtimeDownloadHandler {
	ret := RealtimeDownloadHandler{}
	ret.Downloader = NewHTTPGetDownloader()
	return &ret
}

func (self *RealtimeDownloadHandler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	defer func() {
		if r := recover(); r != nil {
			log.Println(r)
		}
	}()
	w.Header().Set("Content-Encoding", "gzip")
	w.Header().Set("Content-Type", "text/html")
	gz := gzip.NewWriter(w)
	defer gz.Close()
	link := req.FormValue("link")
	linkbyte, err := base64.URLEncoding.DecodeString(link)
	if err != nil {
		gz.Write([]byte(""))
	} else {
		link = string(linkbyte)
		ret := self.ProcessLink(link)
		gz.Write([]byte(ret))
	}
}
