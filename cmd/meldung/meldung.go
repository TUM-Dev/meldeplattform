package main

import (
	"github.com/joschahenningsen/meldeplattform/internal"
	"log"
)

func main() {
	app := internal.NewApp()
	log.Fatal(app.Run())
}
