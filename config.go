package downloader

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"sync"
)

type Config struct {
	DownloaderHost           string   `json:"downloader_host"`
	RedirectorHost           string   `json:"redirector_host"`
	GraphiteHost             string   `json:"graphite_host"`
	SitePatterns             []string `json:"site_patterns"`
	HighPrioritySitePatterns []string `json:"hp_site_patterns"`
	PagePerMinute            int      `json:"page_per_minute"`
	DownloadTimeout          int64    `json:"download_timeout"`
	RedirectChanNum          int      `json:"redirect_chan_num"`
	RedirectChanSize         int      `json:"redirect_chan_size"`
	WritePageFreq            int      `json:"write_page_freq"`
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
		fmt.Println(err)
		return NewDefaultConfig()
	}

	config := Config{}
	err = json.Unmarshal(text, &config)
	if err != nil {
		panic(err)
	}
	return &config
}

var configInstance *Config = nil
var lock sync.Mutex

func ConfigInstance() *Config {
	if configInstance == nil {
		lock.Lock()
		if configInstance == nil {
			configInstance = NewConfig("config.json")
			fmt.Println(configInstance)
		}
		lock.Unlock()
	}
	return configInstance
}
