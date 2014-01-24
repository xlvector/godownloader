package downloader

type BloomFilter struct {
	h    []byte
	size int32
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
	return &bf
}

func (self *BloomFilter) Add(buf string) {
	ha := Hash(buf)
	self.h[ha%self.size] = 1
}

func (self *BloomFilter) Contains(buf string) bool {
	ha := Hash(buf)
	if self.h[ha%self.size] == 1 {
		return true
	} else {
		return false
	}
}
