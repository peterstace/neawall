package main

import (
	"flag"
	"log"
	"net/http"
)

func main() {
	var (
		listenAddrFlag = flag.String("listen", ":8080", "listen address")
		apikeyFlag     = flag.String("apikey", "", "Nearmap API key")
	)
	flag.Parse()

	if *apikeyFlag == "" {
		log.Fatal("--apikey flag not set or empty")
	}

	if err := mainE(*listenAddrFlag, *apikeyFlag); err != nil {
		log.Fatalf("an error occurred: %v", err)
	}
}

func mainE(listenAddr, apikey string) error {
	vertSource := NewTileFetcher(apikey)
	assembler := NewAssembler(vertSource)
	coverage := NewCoverageFetcher(apikey)
	handler := NewHandler(vertSource, assembler, coverage)
	log.Printf("listening on %s", listenAddr)
	return http.ListenAndServe(listenAddr, handler)
}
