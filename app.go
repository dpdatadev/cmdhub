package main

import (
	filerunner "dpdigital/cmdhub/examples/jobs/FileRunner"
	ynews "dpdigital/cmdhub/examples/jobs/HackerNews"
	"log"
)

func init() {
	log.SetPrefix("CMD::>")
	log.SetFlags(0)
	log.Println("RUNNER STARTED")
}

func HackerNewsExample() {
	log.Println("Dumping HTML...")
	ynews.Dump()
	log.Println("Extracting Hacker News Links...")
	ynews.Extract() //TODO, on next example, wrap Rest call function as Command
	log.Println("RUNNER ENDED")
}

func FullExmaple() {
	filerunner.ExecuteHub()
}

func main() {
	//HackerNewsExample()
	log.Println("Running integrated example (no lineage)...")
	FullExmaple()
}
