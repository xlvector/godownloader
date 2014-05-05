package downloader

import (
	"bufio"
	"crawler/downloader/graphite"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"os"
	"strings"
	"time"
)

const PRIORITY_LEVELS = 5

type RedirectorHandler struct {
	metricSender         *graphite.Client
	processedLinks       *BloomFilter
	linksChannel         []chan string
	urlFilter            *URLFilter
	dnsCache             map[string]string
	usedChannels         map[int]int64
	writer               *os.File
	writeCount           int
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
	log.Println("dns lookup", host, ip)
	return ip
}

func (self *RedirectorHandler) Redirect(ci int) {
	priority := ci/ConfigInstance().RedirectChanNum + 1
	n := 0
	for link := range self.linksChannel[ci] {
		n += 1
		log.Println(time.Now().Unix(), "redirector", "send", link)
		pb := PostBody{}
		pb.Links = []string{link}
		jsonBlob, err := json.Marshal(&pb)
		if err == nil {
			req := make(map[string]string)
			req["links"] = string(jsonBlob)
			PostHTTPRequest(ConfigInstance().DownloaderHost, req)
		}
		time.Sleep(60 * time.Second / time.Duration(ConfigInstance().PagePerMinute) / time.Duration(priority))
		if n%100 == 0 {
			time.Sleep(time.Duration(rand.Int63n(120)) * time.Second)
			log.Println("channel sleep : ", ci)
		}
	}
}

func NewRedirectorHandler() *RedirectorHandler {
	ret := RedirectorHandler{}
	ret.metricSender, _ = graphite.New(ConfigInstance().GraphiteHost, "")
	ret.linksChannel = []chan string{}
	for i := 0; i < ConfigInstance().RedirectChanNum*PRIORITY_LEVELS; i++ {
		ret.linksChannel = append(ret.linksChannel, make(chan string, ConfigInstance().RedirectChanSize))
	}
	ret.processedLinks = NewBloomFilter()
	ret.usedChannels = make(map[int]int64)
	ret.urlFilter = NewURLFilter()
	ret.BatchAddLinkFromFile()
	ret.writer, _ = os.Create("links.tsv")
	ret.writeCount = 0
	ret.linksRecvCount = 0
	ret.domainLinksRecvCount = make(map[string]int)
	ret.ticker = time.NewTicker(time.Second * 60)
	go func() {
		for t := range ret.ticker.C {
			log.Println("refresh rules at", t)
			newRules := GetSitePatterns()
			for rule, pri := range newRules {
				log.Println("add rule", rule, "with priority", pri)
				ret.urlFilter.ruleMatcher.AddRule(rule, pri)
			}
		}
	}()

	for i := 0; i < ConfigInstance().RedirectChanNum*PRIORITY_LEVELS; i++ {
		go ret.Redirect(i)
	}
	return &ret
}

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
		self.AddLink(line, "true")
	}
}

func (self *RedirectorHandler) AddLink(link string, isFilter string) {
	log.Println(time.Now().Unix(), "redirector", "receive", link)
	priority := self.Match(link)
	if priority <= 0 {
		return
	}
	addr := ExtractMainDomain(link)
	ci := Hash(addr)%int32(ConfigInstance().RedirectChanNum) + int32((priority-1)*ConfigInstance().RedirectChanNum)
	if len(self.linksChannel[ci]) < ConfigInstance().RedirectChanSize {
		if isFilter != "false" && CheckBloomFilter(link) {
			log.Println("downloaded before : ", link)
			return
		}
		log.Println(time.Now().Unix(), "redirector", "push_queue", link)
		self.processedLinks.Add(link)
		self.linksChannel[ci] <- link
		self.usedChannels[int(ci)] = time.Now().Unix()
	} else {
		if self.writer != nil && rand.Float64() < 0.1 && self.writeCount < 100000 {
			self.writer.WriteString(link + "\n")
			self.writeCount += 1
		}
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
	if len(links) > 0 {
		pb := PostBody{}
		json.Unmarshal([]byte(links), &pb)

		for _, link := range pb.Links {
			self.AddLink(link, isFilter)

			self.linksRecvCount += 1
			if self.urlFilter.Match(link) > 1 {
				domain := ExtractMainDomain(link)
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
