package downloader

import (
	"regexp"
	"strings"
	"math/rand"
)

type Rule struct {
	Regex    *regexp.Regexp
	Priority int
}

type RuleList []Rule

type RuleMatcher struct {
	SiteRules   map[string]RuleList
	CommonRules RuleList
	usedRules map[string]bool
}

func NewRuleMatcher() *RuleMatcher {
	ret := RuleMatcher{}
	ret.SiteRules = make(map[string]RuleList)
	ret.usedRules = make(map[string]bool)
	return &ret
}

func (self *RuleMatcher) AddRule(rule string, priority int) {
     _, ok := self.usedRules[rule]
     if ok {
     return
}
	self.usedRules[rule] = true
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
     if rand.Float64() < 0.00001 {
         newRules := GetNewPatterns()
     	 for rule, pri := range newRules{
     	     self.AddRule(rule, pri)
     	     }
}
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
