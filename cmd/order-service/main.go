package main

import (
	"log"
	"os"
	"test-task/order-service/internal/config"
)

func main() {
	_, err := config.New()
	if err != nil {
		log.Fatal(err)
	}

	os.Exit(0)
}
