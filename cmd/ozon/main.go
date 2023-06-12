package main

import (
	"ozon/rout"
	"os"
	"log"
)

func main() {
	storage := os.Getenv("STORAGE")

	if storage == "in-memory" {
		rout.PostmainInMemory()
	} else if storage == "postgres" {
		rout.Postmain()
	} else {
		log.Fatal("Invalid storage type")
	}
}
