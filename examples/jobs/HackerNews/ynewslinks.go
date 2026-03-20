package ynews

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
)

type LinkExtract struct {
	StatusCode int            `json:"StatusCode"`
	Host       string         `json:"Host"`
	Method     string         `json:"Method"`
	Links      map[string]int `json:"Links"`
}

func Extract() {
	// Connect to "Scrape Serve" microservice https://github.com/dpdatadev/ScrapeServe
	log.Println("Extracting Front Page Links")
	resp, err := http.Get("http://127.0.0.1:7171/links?url=https://news.ycombinator.com")
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()

	var linkExtract LinkExtract
	if err := json.NewDecoder(resp.Body).Decode(&linkExtract); err != nil {
		log.Fatal(err)
	}

	fmt.Println(linkExtract.Links)
}
