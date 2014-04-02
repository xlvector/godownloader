package downloader

import (
	"encoding/json"
	"testing"
)

func Test(t *testing.T) {
	var links LinkConfigArray
	linksJson := []byte(`[
		{"id":6,
		"name":"\u5927\u4f17\u70b9\u8bc4\u5546\u5bb6",
		"pattern":"http:\/\/www.dianping.com\/shop\/[0-9]+",
		"link":"http:\/\/www.dianping.com\/shop\/13878416","priority":1}
	]`)
	err := json.Unmarshal(linksJson, &links)
	if err != nil {
		t.Error(err)
	}

	//t.Error(links)
}
