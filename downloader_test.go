package downloader

import (
	"testing"
)

func TestHTTPGetDownloader(t *testing.T) {
	downloader := NewHTTPGetDownloader()
	html, err := downloader.Download("http://www.sina.com.cn")

	if err != nil {
		t.Error(err)
	}

	if len(html) < 100 {
		t.Error("downloaded page is to small")
	}
}
