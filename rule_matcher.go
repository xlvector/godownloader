package downloader

import (
	"regexp"
	"strings"
)

type Rule struct {
	Regex    *regexp.Regexp
	Priority int
}

type RuleList []Rule

type RuleMatcher struct {
	SiteRules   map[string]RuleList
	CommonRules RuleList
}

func NewRuleMatcher() *RuleMatcher {
	ret := RuleMatcher{}
	ret.SiteRules = make(map[string]RuleList)
	return &ret
}

func (self *RuleMatcher) AddRule(rule string, priority int) {
	domain := ExtractMainDomain(rule)
	pattern := regexp.MustCompile(rule)
	if strings.Contains(domain, "*") || strings.Contains(domain, "[") {
		self.CommonRules = append(self.CommonRules, Rule{Regex: pattern, Priority: priority})
	} else {
		_, ok := self.SiteRules[domain]
		if !ok {
			self.SiteRules[domain] = []Rule{}
		}
		self.SiteRules[domain] = append(self.SiteRules[domain], Rule{Regex: pattern, Priority: priority})
	}
}

func (self *RuleMatcher) MatchRule(link string) int {
	domain := ExtractMainDomain(link)
	rules, ok := self.SiteRules[domain]
	maxPriority := 0
	if ok {
		for _, rule := range rules {
			if rule.Regex.FindString(link) == link {
				if maxPriority < rule.Priority {
					maxPriority = rule.Priority
				}
			}
		}
	}

	if maxPriority <= 0 {
		for _, rule := range self.CommonRules {
			if rule.Regex.FindString(link) == link {
				if maxPriority < rule.Priority {
					maxPriority = rule.Priority
				}
			}
		}
	}
	return maxPriority
}
