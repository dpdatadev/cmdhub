package main

import (
	ynews "dpdigital/cmdhub/examples/jobs/HackerNews"
	"log"
)

func init() {
	log.SetPrefix("CMD::>")
	log.SetFlags(0)
	log.Println("RUNNER STARTED")
}

func main() {
	log.Println("Dumping HTML...")
	ynews.Dump()
	log.Println("Extracting Links from the Front/Home Page...")
	ynews.Extract()
	log.Println("RUNNER ENDED")
}
