package downloader

import (
	"testing"
)

func TestIsUTF8(t *testing.T) {
	if IsUTF8("hello world") == false {
		t.Error()
	}

	if IsUTF8("c今年天天aaaa气bbbb") == false {
		t.Error()
	}
}

func TestChinesePage(t *testing.T) {
	if IsChinesePage("今天的天气很好") == false {
		t.Error()
	}

	if IsChinesePage("today is a good day") == true {
		t.Error()
	}
}
