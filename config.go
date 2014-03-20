package downloader

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"sync"
	"time"
)

type Config struct {
	DownloaderHost           string   `json:"downloader_host"`
	RedirectorHost           string   `json:"redirector_host"`
	FilterHost               string   `json:"filter_host"`
	GraphiteHost             string   `json:"graphite_host"`
	SitePatterns             []string `json:"site_patterns"`
	HighPrioritySitePatterns []string `json:"hp_site_patterns"`
	PagePerMinute            int      `json:"page_per_minute"`
	DownloadTimeout          int64    `json:"download_timeout"`
	RedirectChanNum          int      `json:"redirect_chan_num"`
	RedirectChanSize         int      `json:"redirect_chan_size"`
	WritePageFreq            int      `json:"write_page_freq"`
}

type LinkConfig struct {
	id       int
	name     string
	pattern  string
	link     string
	priority int
}

func NewDefaultConfig() *Config {
	config := Config{
		PagePerMinute:   10,
		DownloadTimeout: 10,
		RedirectChanNum: 10,
	}
	return &config
}

func NewConfig(path string) *Config {
	text, err := ioutil.ReadFile(path)
	if err != nil {
		log.Println(err)
		return NewDefaultConfig()
	}

	config := Config{}
	err = json.Unmarshal(text, &config)
	if err != nil {
		panic(err)
	}

	downloader := NewHTTPGetProxyDownloader("http://10.181.10.21")
	linksJson, err := downloader.Download("http://10.105.75.102/pagemining-tools/links/list.php")
	links := []LinkConfig{}
	err = json.Unmarshal([]byte(linksJson), &links)
	if err != nil {
		panic(err)
	}
	pb := PostBody{}
	pb.Links = []string{}

	for _, link := range links {
		log.Println("links-tool", link)
		pb.Links = append(pb.Links, link.link)
		if link.priority == 1 {
			config.HighPrioritySitePatterns = append(config.HighPrioritySitePatterns, link.pattern)
		} else if link.priority == 2 {
			config.SitePatterns = append(config.SitePatterns, link.pattern)
		}
	}
	jsonBlob, err := json.Marshal(&pb)
	if err == nil {
		req := make(map[string]string)
		req["links"] = string(jsonBlob)
		PostHTTPRequest(ConfigInstance().RedirectorHost, req)
	}
	return &config
}

var configInstance *Config = nil
var lock sync.Mutex
var lastLoadConfigTime int64

func ConfigInstance() *Config {
	now := time.Now().Unix()
	if configInstance == nil || (now-lastLoadConfigTime) > 60 {
		lock.Lock()
		if configInstance == nil || (now-lastLoadConfigTime) > 60 {
			configInstance = NewConfig("config.json")
			log.Println("load config file")
			lastLoadConfigTime = time.Now().Unix()
		}
		lock.Unlock()
	}
	return configInstance
}
