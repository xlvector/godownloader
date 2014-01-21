package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"time"
)

func call() int64 {
	host := flag.String("host", "http://localhost:8113/download", "host")
	link := flag.String("link", "", "link to download")
	flag.Parse()

	root := *host

	jsonBlob := "{\"links\": []}"

	if len(*link) > 0 {
		jsonBlob = "{\"links\": [\"" + *link + "\"]}"
	}

	post := url.Values{}
	post.Set("links", jsonBlob)

	start := time.Now()
	resp, err := http.PostForm(root, post)

	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()
	content, err := ioutil.ReadAll(resp.Body)
	fmt.Println(string(content))
	if err != nil {
		panic(err)
	}
	end := time.Now()

	return end.Sub(start).Nanoseconds()
}

func main() {
	call()
}
