package downloader

import (
	"testing"
)

func TestRuleMatcher(t *testing.T) {
	rm := NewRuleMatcher()

	rm.AddRule("http://[a-z0-9]+.sina.com.cn[/]*", 2)
	rm.AddRule("http://[a-z0-9]+.[a-z0-9]+.com/", 1)

	if rm.MatchRule("http://t.sina.com.cn/") != 2 {
		t.Error()
	}

	if rm.MatchRule("http://www.ustc.com/") != 1 {
		t.Error()
	}

	if rm.MatchRule("http://www.sina.com.cn/hello") != 0 {
		t.Error()
	}
}
