package downloader

import (
	"crawler/downloader/graphite"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"strings"
	"time"
)

const PRIORITY_LEVELS = 6

type RedirectorHandler struct {
	metricSender         *graphite.Client
	processedLinks       *BloomFilter
	linksChannel         []chan Link
	urlFilter            *URLFilter
	dnsCache             map[string]string
	usedChannels         map[int]int64
	linksRecvCount       int
	domainLinksRecvCount map[string]int
	ticker *time.Ticker
}

func (self *RedirectorHandler) Match(link string) int {
	return self.urlFilter.Match(link)
}

func (self *RedirectorHandler) GetIP(host string) string {
	ip, ok := self.dnsCache[host]
	if ok {
		return ip
	}
	ip = LoopUpIp(host)
	if ip == "" {
		ip = host
	}
	return ip
}

func (self *RedirectorHandler) Redirect(ci int) {
	priority := ci/ConfigInstance().RedirectChanNum + 1
	n := 0
	for link := range self.linksChannel[ci] {
		n += 1
		log.Println(time.Now().Unix(), "redirector", "send", link.LinkURL, priority)
		query := extractSearchQuery(link.LinkURL)
		if len(query) > 0 {
			setStatus(query, "redirector.send." + ExtractDomainOnly(link.LinkURL))
		}
		pb := PostBody{}
		pb.Links = []Link{link}
		jsonBlob, err := json.Marshal(&pb)
		if err == nil {
			req := make(map[string]string)
			req["links"] = string(jsonBlob)
			PostHTTPRequest(ConfigInstance().DownloaderHost, req)
		}
		time.Sleep(time.Duration(int64(time.Second) * 2 / int64(1 + priority)))
		if n%200 == 0 {
			time.Sleep(time.Duration(rand.Int63n(30) / int64(1 + priority)) * time.Second)
		}
	}
}

func NewRedirectorHandler() *RedirectorHandler {
	ret := RedirectorHandler{}
	ret.metricSender, _ = graphite.New(ConfigInstance().GraphiteHost, "")
	ret.linksChannel = []chan Link{}
	for i := 0; i < ConfigInstance().RedirectChanNum*PRIORITY_LEVELS; i++ {
		ret.linksChannel = append(ret.linksChannel, make(chan Link, ConfigInstance().RedirectChanSize))
	}
	ret.processedLinks = NewBloomFilter()
	ret.usedChannels = make(map[int]int64)
	ret.urlFilter = NewURLFilter()
	ret.linksRecvCount = 0
	ret.domainLinksRecvCount = make(map[string]int)
	ret.ticker = time.NewTicker(time.Second * 60)
	go func() {
		for t := range ret.ticker.C {
			newRules := GetSitePatterns()
			for rule, pri := range newRules {
				ret.urlFilter.ruleMatcher.AddRule(rule, pri)
			}
			log.Println(t)
		}
	}()

	for i := 0; i < ConfigInstance().RedirectChanNum*PRIORITY_LEVELS; i++ {
		go ret.Redirect(i)
	}
	return &ret
}
/*
func (self *RedirectorHandler) BatchAddLinkFromFile() {
	f, err := os.Open("links.tsv")
	defer f.Close()
	if err != nil {
		return
	}
	reader := bufio.NewReader(f)
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			break
		}
		self.AddLink(line, "true", "normal")
	}
}
*/
func IsSearchEnginePage(link string) bool {
	if strings.Contains(link, "http://www.baidu.com/s?word="){
		return true
	}
	if strings.Contains(link, "http://www.sogou.com/"){
		return true
	}
	if strings.Contains(link, "http://www.so.com/"){
		return true
	}
	return false
}

func (self *RedirectorHandler) AddLink(link Link, isFilter string, pri string) {
	priority := self.Match(link.LinkURL)

	isSePage := IsSearchEnginePage(link.Referrer)	
	if isSePage {
		priority = PRIORITY_LEVELS - 1
		log.Println("redirector search engine referrer", link)
	}
	if pri == "high" {
		priority = PRIORITY_LEVELS
	}
	if priority <= 0 {
		return
	}
	
	
	addr := ExtractMainDomain(link.LinkURL)
	ci := Hash(addr)%int32(ConfigInstance().RedirectChanNum) + int32((priority-1)*ConfigInstance().RedirectChanNum)
	if len(self.linksChannel[ci]) < ConfigInstance().RedirectChanSize {
		if isFilter != "false" && CheckBloomFilter(link.LinkURL) {
			return
		}
		//query := extractSearchQuery(link.LinkURL)
		self.processedLinks.Add(link.LinkURL)
		self.linksChannel[ci] <- link
		self.usedChannels[int(ci)] = time.Now().Unix()
	}
}

func (self *RedirectorHandler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	defer func() {
		if r := recover(); r != nil {
			log.Println(r)
		}
	}()

	links := req.PostFormValue("links")
	isFilter := req.PostFormValue("filter")
	priority := req.PostFormValue("priority")
	if len(links) > 0 {
		pb := PostBody{}
		json.Unmarshal([]byte(links), &pb)

		for _, link := range pb.Links {
			self.AddLink(link, isFilter, priority)

			self.linksRecvCount += 1
			if self.urlFilter.Match(link.LinkURL) > 1 {
				domain := ExtractMainDomain(link.LinkURL)
				domain = strings.Replace(domain, ".", "_", -1)
				self.domainLinksRecvCount[domain] += 1
			}
		}
	}

	if rand.Float64() < 0.2 {
		linkChannelTotalSize := 0
		maxChannelSize := 0
		nonEmptyQueueCount := 0

		for _, cn := range self.linksChannel {

			size := len(cn)

			linkChannelTotalSize += size

			if maxChannelSize < size {
				maxChannelSize = size
			}
			if size > 0 {
				nonEmptyQueueCount += 1
			}
		}

		usedChannelCount := 0
		currentTm := time.Now().Unix()
		for _, tm := range self.usedChannels {
			if currentTm-tm > 120 {
				continue
			}
			usedChannelCount += 1
		}
		self.metricSender.Gauge("crawler.redirector."+GetHostName()+"."+Port+".channelsize", int64(linkChannelTotalSize), 1.0)
		self.metricSender.Gauge("crawler.redirector."+GetHostName()+"."+Port+".maxchannelsize", int64(maxChannelSize), 1.0)
		self.metricSender.Gauge("crawler.redirector."+GetHostName()+"."+Port+".nonemptychannelcount", int64(nonEmptyQueueCount), 1.0)
		self.metricSender.Gauge("crawler.redirector."+GetHostName()+"."+Port+".usedchannelcount", int64(usedChannelCount), 1.0)
		self.metricSender.Gauge("crawler.redirector."+GetHostName()+"."+Port+".linksRecvCount", int64(self.linksRecvCount), 1.0)
		for domain, recvcount := range self.domainLinksRecvCount {
			metricName := "crawler.redirector." + GetHostName() + "." + Port + ".domainLinksRecvCount." + domain
			self.metricSender.Gauge(metricName, int64(recvcount), 1.0)
		}
	}
	fmt.Fprint(w, "thanks for your links")
}
