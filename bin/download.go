package main

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"time"
)

func main() {
	client := &http.Client{
		Transport: &http.Transport{
			DisableKeepAlives:     true,
			ResponseHeaderTimeout: 10 * time.Second,
		},
	}

	req, err := http.NewRequest("GET", "http://www.baidu.com/link?url=pHvjcd6ZcNE-aA1L3WsBrTapkz81_vnmJ9fCxXCGQuqoH38-sRwhHSxQjFkCxUCz26hnCufO9DVrfj2FaQP3Jq", nil)
	if err != nil || req == nil || req.Header == nil {
		return
	}
	resp, err := client.Do(req)
	fmt.Println(resp.Request.URL.String())
	if err != nil || resp == nil || resp.Body == nil {
		return
	} else {
		defer resp.Body.Close()

		html, _ := ioutil.ReadAll(resp.Body)
		fmt.Println(string(html))
	}
}
