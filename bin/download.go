package main

import (
	"fmt"
	//"io/ioutil"
	//"net/http"
	"net"
	//"time"
)

/*
func dialTimeout(network, addr string) (net.Conn, error) {
	timeout := time.Duration(ConfigInstance().DownloadTimeout) * time.Second
	deadline := time.Now().Add(timeout)
	c, err := net.DialTimeout(network, addr, timeout)
	if err != nil {
		return nil, err
	}
	c.SetDeadline(deadline)
	return c, nil
}
*/

func main() {
	/*
		client := &http.Client{
			Transport: &http.Transport{
				Dial:                  dialTimeout,
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
	*/
	addrs, _ := net.LookupHost("blog.xlvector.net")
	fmt.Println(addrs)
}
