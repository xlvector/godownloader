package downloader

import (
	"testing"
)

func TestCleanHTML(t *testing.T) {
	html := `
		<html>
			<head>
				<title>Hello World</title>
			</head>
			<body>
				<a href="#">Hello          World!</a>
			</body>
		</html>
	`
	c := NewHTMLCleaner()
	ret := c.CleanHTML([]byte(html))

	if string(ret) != "<html> <head> <title>Hello World</title> </head> <body> <a href=\"#\">Hello World!</a> </body> </html> " {
		t.Error("clean html failed", ret)
	}
}

func TestToUTF8(t *testing.T) {
	html := `<html>
				<head>
					<title>字符测试</title>
				</head>
				<body>
					字符测试
				</body>
			</html>`

	cleaner := NewHTMLCleaner()

	ret := cleaner.ToUTF8([]byte(html))

	if ret == nil {
		t.Error("ToUTF8 failed", ret)
	}
}
