package downloader

import (
	"encoding/json"
	"io/ioutil"
	"sync"
)

type Config struct {
	DownloaderHost string   `json:"downloader_host"`
	RedirectorHost string   `json:"redirector_host"`
	SitePatterns   []string `json:"site_patterns"`
}

func NewConfig(path string) *Config {
	text, err := ioutil.ReadFile(path)
	if err != nil {
		panic(err)
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
		}
		lock.Unlock()
	}
	return configInstance
}
