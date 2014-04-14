package downloader

type URLFilter struct {
	ruleMatcher *RuleMatcher
}

func NewURLFilter() *URLFilter {
	ret := URLFilter{}
	ret.ruleMatcher = NewRuleMatcher()
	return &ret
}

func (self *URLFilter) Match(link string) int {
	return self.ruleMatcher.MatchRule(link)
}
