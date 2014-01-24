package downloader

import (
	"testing"
)

func TestExtractLinks(t *testing.T) {
	html := `
		<html>
			<head>
				<title>Hello World</title>
			</head>
			<body>
				<a href="../abc">Hello          World!</a>
				<a href=..>aaa</a>
				<span>
					<a href="/def">DEF</a>
				</span>
				<a HrEF="http://ddd"></a>
			</body>
		</html>
	`
	links := ExtractLinks([]byte(html), "http://xlvector.net/blog")
	d := make(map[string]bool)
	for _, link := range links {
		d[link] = true
	}
	for _, link := range []string{"http://xlvector.net", "http://xlvector.net/abc", "http://xlvector.net/def", "http://ddd"} {
		_, ok := d[link]
		if !ok {
			t.Error(link)
			t.Error(d)
		}
	}

	if ExtractDomain("https://www.sina.com/aaa/bbb") != "https://www.sina.com" {
		t.Error()
	}

	if ExtractMainDomain("http://www.sina.com.cn") != "sina.com.cn" {
		t.Error()
	}

	if ExtractMainDomain("http://sina.com") != "sina.com" {
		t.Error()
	}
}
