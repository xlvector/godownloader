package downloader

import (
	"math/rand"
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
	usedRules   map[string]bool
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

type LinkConfig struct {
	Id       int    `json:"id"`
	Name     string `json:"name"`
	Pattern  string `json:"pattern"`
	Link     string `json:"link"`
	Priority int    `json:"priority"`
}
type LinkConfigArray []LinkConfig

func GetNewPatterns() map[string]int {
	log.Println("addlinkconfig")
	downloader := NewDefaultHTTPGetProxyDownloader("http://10.181.10.21")
	linksJson, err := downloader.Download("http://10.105.75.102/pagemining-tools/links/list.php")
	if err != nil {
		log.Println("addlinkconfig", err)
	}
	ret := make(map[string]int)
	var links LinkConfigArray
	err = json.Unmarshal([]byte(linksJson), &links)
	if err != nil {
		log.Println(err)
		return ret
	}
	log.Println(links)
	pb := PostBody{}
	pb.Links = []string{}
	for _, link := range links {
		pb.Links = append(pb.Links, link.Link)
		ret[link.Pattern] = link.Priority
	}
	jsonBlob, err := json.Marshal(&pb)
	if err == nil {
		req := make(map[string]string)
		req["links"] = string(jsonBlob)
		PostHTTPRequest(ConfigInstance().DownloaderHost, req)
	}
	return ret
}

func (self *RuleMatcher) MatchRule(link string) int {
	if rand.Float64() < 0.00001 {
		newRules := GetNewPatterns()
		for rule, pri := range newRules {
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
