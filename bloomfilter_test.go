package downloader

import (
	"strconv"
	"testing"
)

func TestBloomFilter(t *testing.T) {
	if GetDayTimeStamp() < 60 {
		t.Error("current day timestamp", GetDayTimeStamp())
	}

	bf := NewBloomFilter()

	for i := 0; i < 1000000; i++ {
		buf := strconv.FormatInt((int64)(i), 10)
		if i%2 == 0 {
			bf.Add(buf)
		}
	}

	n := 0
	for i := 0; i < 1000000; i++ {
		buf := strconv.FormatInt((int64)(i), 10)
		if i%2 == 0 {
			if !bf.Contains(buf) {
				t.Error()
			}
		} else {
			if bf.Contains(buf) {
				n += 1
			}
		}
	}
	if n > 2000 {
		t.Error(n)
	}
}
