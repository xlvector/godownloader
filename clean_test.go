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
