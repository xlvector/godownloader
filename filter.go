package downloader

import (
	"regexp"
)

type URLFilter struct {
	patterns  []*regexp.Regexp
	hpatterns []*regexp.Regexp
}

func NewURLFilter() *URLFilter {
	ret := URLFilter{}
	ret.patterns = []*regexp.Regexp{}
	ret.hpatterns = []*regexp.Regexp{}
	for _, pt := range ConfigInstance().SitePatterns {
		re := regexp.MustCompile(pt)
		ret.patterns = append(ret.patterns, re)
	}
	for _, pt := range ConfigInstance().HighPrioritySitePatterns {
		re := regexp.MustCompile(pt)
		ret.hpatterns = append(ret.hpatterns, re)
	}
	return &ret
}

func (self *URLFilter) Match(link string) int {
	for _, pt := range self.hpatterns {
		if pt.FindString(link) == link {
			return 2
		}
	}
	for _, pt := range self.patterns {
		if pt.FindString(link) == link {
			return 1
		}
	}
	return 0
}
