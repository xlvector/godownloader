package downloader

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"sync"
	"time"
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
var lastLoadConfigTime int64
var lock sync.Mutex

func ConfigInstance() *Config {
	if lastLoadConfigTime == 0 {
		lastLoadConfigTime = time.Now().Unix()
	}
	if configInstance == nil || (time.Now().Unix()-lastLoadConfigTime) > 60 {
		lock.Lock()
		if configInstance == nil {
			configInstance = NewConfig("config.json")
			fmt.Println("reload config")
		}
		lock.Unlock()
	}
	return configInstance
}
