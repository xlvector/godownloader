package main

import (
	"crawler/downloader"
	"flag"
	"log"
	"net/http"
	"runtime"
	"time"
)

func main() {
	runtime.GOMAXPROCS(4)
	port := flag.String("port", "8100", "port number")
	flag.Parse()

	downloader.Port = *port

	http.Handle("/download", downloader.NewDownloadHanler())
	http.Handle("/redirect", downloader.NewRedirectorHandler())

	s := &http.Server{
		Addr:           ":" + *port,
		ReadTimeout:    30 * time.Second,
		WriteTimeout:   30 * time.Second,
		MaxHeaderBytes: 1 << 20,
	}
	log.Fatal(s.ListenAndServe())
}
