package main

import (
	"crawler/downloader"
	"flag"
	"log"
	"net/http"
	"time"
)

func main() {
	port := flag.String("port", "8113", "port number")
	flag.Parse()

	http.Handle("/download", downloader.NewDownloadHanler())

	s := &http.Server{
		Addr:           ":" + *port,
		ReadTimeout:    30 * time.Second,
		WriteTimeout:   30 * time.Second,
		MaxHeaderBytes: 1 << 20,
	}
	log.Fatal(s.ListenAndServe())
}
