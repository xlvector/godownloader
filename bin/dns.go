package main

import (
	"flag"
	"github.com/soundcloud/go-dns-resolver/resolv"
	"log"
)

var (
	name  = flag.String("name", "example.com", "name to resolv")
	qType = flag.String("type", "A", "name to resolv")
)

func init() {
	flag.Parse()
}

func main() {
	answer, err := resolv.LookupString(*qType, *name)
	if err != nil {
		log.Fatalf("Couldn't resolve name: %s", err)
	}
	log.Printf("Answer: %v", answer)
}