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
	log.Println("Extracting Hacker News Links...")
	ynews.Extract() //TODO, on next example, wrap Rest call function as Command
	log.Println("RUNNER ENDED")
}
