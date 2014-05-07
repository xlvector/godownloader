package downloader

import (
	"code.google.com/p/mahonia"
	"github.com/saintfish/chardet"
	"strings"
)

type HTMLCleaner struct {
	detector *chardet.Detector
	gbk      mahonia.Decoder
}

func NewHTMLCleaner() *HTMLCleaner {
	ret := HTMLCleaner{}
	ret.detector = chardet.NewHtmlDetector()
	ret.gbk = mahonia.NewDecoder("gb18030")
	return &ret
}

func (self *HTMLCleaner) detectCharset(html []byte) string {
	ret, err := self.detector.DetectBest(html)
	if err != nil {
		return ""
	}
	return strings.ToLower(ret.Charset)
}

func (self *HTMLCleaner) CleanHTML(src []byte) []byte {
	dst := []byte{}

	prev := byte(0)
	for i, ch := range src {
		if ch <= 32 {
			if i > 0 && prev > 32 {
				dst = append(dst, 32)
				prev = ch
			}
		} else {
			dst = append(dst, ch)
			prev = ch
		}
	}
	return dst
}

func (self *HTMLCleaner) ToUTF8(html []byte) []byte {
	charset := self.detectCharset(html)
	if !strings.Contains(charset, "gb"){
		charset = "utf-8"
	}
	if charset == "utf-8" || charset == "utf8" {
		return html
	} else if charset == "gb2312" || charset == "gb-2312" || charset == "gbk" || charset == "gb18030" || charset == "gb-18030" {
		ret, ok := self.gbk.ConvertStringOK(string(html))
		if ok {
			return []byte(ret)
		} else {
			return nil
		}
	} else {
		return nil
	}
}
