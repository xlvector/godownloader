package main

import (
	"fmt"
	"crawler/downloader"
)

func main() {
	dl := downloader.NewHTTPGetDownloader()
	html, _, _ := dl.Download("http://www.baidu.com/s?word=13912510606")
	fmt.Println(html)
}
