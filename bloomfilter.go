package downloader

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
)

type BloomFilter struct {
	h    []byte
	size int32
	hit  int64
}

func Hash(buf string) int32 {
	var seed int32
	var h int32

	seed = 131
	h = 0

	for _, ch := range buf {
		h = h*seed + int32(ch)
	}

	if h < 0 {
		h *= -1
	}
	return h
}

func NewBloomFilter() *BloomFilter {
	bf := BloomFilter{}
	bf.size = 100000000
	bf.h = make([]byte, bf.size)

	for i := int32(0); i < bf.size; i++ {
		bf.h[i] = 0
	}
	bf.Load()
	return &bf
}

func (self *BloomFilter) Save() {
	f, err := os.Create("filter.data")
	if err != nil {
		fmt.Println(err)
	}
	defer f.Close()

	for i, val := range self.h {
		if val == 1 {
			f.WriteString(strconv.Itoa(i) + "\n")
		}
	}
}

func (self *BloomFilter) Load() {
	f, err := os.Open("filter.data")
	if err != nil {
		fmt.Println(err)
	}
	defer f.Close()

	r := bufio.NewReader(f)
	n := 0
	for {
		line, err := r.ReadString('\n')
		if err != nil {
			break
		}
		ha, err := strconv.Atoi(line)
		if err != nil {
			fmt.Println(err)
		}
		self.h[ha] = 1
		n += 1
	}
	fmt.Println("lines", n)
	n = 0
	for _, v := range self.h {
		n += int(v)
	}
	fmt.Println("lines", n)
}

func (self *BloomFilter) Add(buf string) {
	ha := Hash(buf)
	self.h[ha%self.size] = 1
}

func (self *BloomFilter) Contains(buf string) bool {
	self.hit += 1
	if self.hit%1000 == 0 {
		self.Save()
	}
	ha := Hash(buf)
	if self.h[ha%self.size] == 1 {
		return true
	} else {
		return false
	}
}
