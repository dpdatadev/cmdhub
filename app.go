package main

import "log"

func init() {
	log.SetPrefix("CMD::>")
	log.SetFlags(0)
	log.Println("RUNNER STARTED")
}

func main() {
	GetHackerNews()
	log.Println("RUNNER ENDED")
}
