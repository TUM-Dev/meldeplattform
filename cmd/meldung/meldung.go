package main

import (
	"github.com/TUM-Dev/meldeplattform/internal"
	"log"
)

func main() {
	app := internal.NewApp()
	log.Fatal(app.Run())
}
