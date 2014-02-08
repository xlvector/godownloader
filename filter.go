package downloader

import (
	"fmt"
)

type URLFilter struct {
	ruleMatcher *RuleMatcher
}

func NewURLFilter() *URLFilter {
	ret := URLFilter{}
	ret.ruleMatcher = NewRuleMatcher()
	for _, pt := range ConfigInstance().SitePatterns {
		ret.ruleMatcher.AddRule(pt, 1)
	}
	for _, pt := range ConfigInstance().HighPrioritySitePatterns {
		ret.ruleMatcher.AddRule(pt, 2)
	}
	fmt.Println(ret.ruleMatcher)
	return &ret
}

func (self *URLFilter) Match(link string) int {
	return self.ruleMatcher.MatchRule(link)
}
