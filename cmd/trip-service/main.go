package main

import (
	"fmt"
	"log"

	"github.com/Ksusha-Pushkova/trip_go/internal/config"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	fmt.Printf("%+v\n", cfg)
}