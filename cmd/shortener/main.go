package main

import (
	"fmt"
	"github.com/MalyginaEkaterina/shortener/internal/app"
)

var (
	buildVersion string
	buildDate    string
	buildCommit  string
)

func main() {
	if buildVersion != "" {
		fmt.Printf("Build version: %s\n", buildVersion)
	} else {
		fmt.Printf("Build version: N/A\n")
	}
	if buildDate != "" {
		fmt.Printf("Build date: %s\n", buildDate)
	} else {
		fmt.Printf("Build date: N/A\n")
	}
	if buildCommit != "" {
		fmt.Printf("Build commit: %s\n", buildCommit)
	} else {
		fmt.Printf("Build commit: N/A\n")
	}
	app.Start()
}

//user=postgres password=12345 host=localhost port=5432 dbname=postgres
