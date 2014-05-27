package main

import (
	"os"
	"crawler/downloader"
	"unicode/utf8"
	"strconv"
	"strings"
	"bufio"
	"flag"
	"time"
)

func WritePage(writer *os.File, page downloader.WebPage) {
	if !utf8.ValidString(page.Link) {
		return
	}

	if !utf8.ValidString(page.Html) {
		return
	}

	writer.WriteString(strconv.FormatInt(page.DownloadedAt, 10))
	writer.WriteString("\t")
	writer.WriteString(page.Link)
	writer.WriteString("\t")
	writer.WriteString(page.Html)
	writer.WriteString("\t")
	writer.WriteString(page.RespInfo)
	writer.WriteString("\n")
}

func BatchDownload(linkFile string) {
	f, err := os.Open(linkFile)
	if err != nil {
		return
	}
	r := bufio.NewReader(f)
	dl := downloader.NewHTTPGetDownloader()
	writer, _ := os.Create(strconv.FormatInt(time.Now().UnixNano(), 10) + ".tsv")
	for {
		line, err := r.ReadString('\n')
		if err != nil {
			break
		}
		line = strings.Trim(line, "\n")
		html := ""
		resp := ""
		html, resp, err = dl.Download(line)
		if err == nil {
			page := downloader.WebPage{Link: line, Html: html, RespInfo: resp, DownloadedAt: time.Now().Unix()}
			WritePage(writer, page)
		}
	}
	writer.Close()
}

func main() {
	file := flag.String("file", "", "file")
	flag.Parse()

	BatchDownload(*file)
}