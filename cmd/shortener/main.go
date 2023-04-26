package main

import (
	"fmt"
	"github.com/MalyginaEkaterina/shortener/internal/app"
)

var (
	buildVersion = "N/A"
	buildDate    = "N/A"
	buildCommit  = "N/A"
)

func main() {
	fmt.Printf("Build version: %s\n", buildVersion)
	fmt.Printf("Build date: %s\n", buildDate)
	fmt.Printf("Build commit: %s\n", buildCommit)
	app.Start()
}

//user=postgres password=12345 host=localhost port=5432 dbname=postgres
