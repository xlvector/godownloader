package downloader

import (
	"testing"
)

func TestHTTPGetDownloader(t *testing.T) {
	downloader := NewHTTPGetDownloader()
	html, resp, err := downloader.Download("http://www.sina.com.cn")

	if err != nil {
		t.Error(err)
	}

	if len(html) < 50 {
		t.Error("downloaded page is to small")
	}

	if len(resp) < 50 {
		t.Error("Response page is to small, ", resp)
	}
}
