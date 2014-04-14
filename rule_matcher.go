package downloader

import (
	"encoding/json"
	"log"
	"regexp"
	"strings"
	"time"
)

type Rule struct {
	Regex    *regexp.Regexp
	Priority int
}

type RuleList []Rule

type RuleMatcher struct {
	SiteRules    map[string]RuleList
	CommonRules  RuleList
	usedRules    map[string]bool
	lastRefreshTime int64
}

func NewRuleMatcher() *RuleMatcher {
	ret := RuleMatcher{}
	ret.SiteRules = make(map[string]RuleList)
	ret.usedRules = make(map[string]bool)
	ret.lastRefreshTime = time.Now().Unix()
	newRules := GetSitePatterns()
	for rule, pri := range newRules {
		log.Println("add rule", rule, "with priority", pri)
		ret.AddRule(rule, pri)
	}
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
	Id        int    `json:"id"`
	Name      string `json:"name"`
	Pattern   string `json:"pattern"`
	Link      string `json:"link"`
	Priority  int    `json:"priority"`
	Entrance1 string `json:"entrance_1"`
	Entrance2 string `json:"entrance_2"`
	Entrance3 string `json:"entrance_3"`
}
type LinkConfigArray []LinkConfig

type TemplateConfig struct {
	Pattern string `json:"_pattern"`
	Link    string `json:"_link"`
}

type TemplateConfigArray []TemplateConfig

func GetSitePatterns() map[string]int {
	log.Println("addlinkconfig")
	downloader := NewDefaultHTTPGetProxyDownloader("http://10.181.10.21")
	linksJson, _, err := downloader.Download("http://10.105.75.102/pagemining-tools/links/list.php")
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
	downloadPb := PostBody{}
	redirectPb := PostBody{}
	downloadPb.Links = []string{}
	redirectPb.Links = []string{}
	for _, link := range links {
		redirectPb.Links = append(redirectPb.Links, link.Link)
		if len(link.Entrance1) > 0 {
			downloadPb.Links = append(downloadPb.Links, link.Entrance1)
		}
		if len(link.Entrance2) > 0 {
			downloadPb.Links = append(downloadPb.Links, link.Entrance2)
		}
		if len(link.Entrance3) > 0 {
			downloadPb.Links = append(downloadPb.Links, link.Entrance3)
		}
		ret[link.Pattern] = link.Priority
	}

	linksJson, _, err = downloader.Download("http://10.105.75.102/pagemining-tools/template_json.php")
	if err != nil {
		log.Println("addlinkconfig", err)
	}
	var templateLinks TemplateConfigArray
	err = json.Unmarshal([]byte(linksJson), &templateLinks)
	if err != nil {
		log.Println(err)
		return ret
	}
	for _, link := range templateLinks {
		redirectPb.Links = append(redirectPb.Links, link.Link)
		ret[link.Pattern] = 2
	}

	jsonBlob, err := json.Marshal(&redirectPb)
	if err == nil {
		req := make(map[string]string)
		req["links"] = string(jsonBlob)
		PostHTTPRequest(ConfigInstance().RedirectorHost, req)
	}

	jsonBlob, err = json.Marshal(&downloadPb)
	if err == nil {
		req := make(map[string]string)
		req["links"] = string(jsonBlob)
		PostHTTPRequest(ConfigInstance().DownloaderHost, req)
	}
	return ret
}

func (self *RuleMatcher) MatchRule(link string) int {
	if(time.Now().Unix() - self.lastRefreshTime > 60) {
		log.Println("refresh rules at", time.Now())
		newRules := GetSitePatterns()
		for rule, pri := range newRules {
			log.Println("add rule", rule, "with priority", pri)
			self.AddRule(rule, pri)
		}
		self.lastRefreshTime = time.Now().Unix()
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
