package downloader

import (
	"log"
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
	return &ret
}

func (self *URLFilter) Match(link string) int {s
	return self.ruleMatcher.MatchRule(link)
}
