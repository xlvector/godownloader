package downloader

import (
	"encoding/json"
	"io/ioutil"
	"log"
)

type Config struct {
	DownloaderHost           string   `json:"downloader_host"`
	HighPriorityDLHost	string `json:"high_priority_downloader_host"`
	LowPriorityDLHost	string `json:"low_priority_downloader_host"`
	RedirectorHost           string   `json:"redirector_host"`
	FilterHost               string   `json:"filter_host"`
	GraphiteHost             string   `json:"graphite_host"`
	PagePerMinute            int      `json:"page_per_minute"`
	DownloadTimeout          int64    `json:"download_timeout"`
	RedirectChanNum          int      `json:"redirect_chan_num"`
	RedirectChanSize         int      `json:"redirect_chan_size"`
	WritePageFreq            int      `json:"write_page_freq"`
	SitePatterns			 map[string]int
	ExtractLinks			int `json:"extract_links"`
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
	return &config
}

var configInstance *Config = NewConfig("config.json")

func ConfigInstance() *Config {
	return configInstance
}
