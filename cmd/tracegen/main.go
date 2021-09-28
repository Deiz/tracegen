package main

import (
	"log"
	"os"

	"github.com/Deiz/tracegen"
)

func main() {
	settings := tracegen.DefaultSettings()
	settings.Exclude = append(settings.Exclude, `/generated(/|$)`)

	flags := tracegen.DefaultFlags(&settings)

	if err := flags.Parse(os.Args[1:]); err != nil {
		log.Fatalf("failed to parse flags: %v", err)
	}

	if flags.NArg() < 1 {
		log.Fatal("must specify at least one pattern")
	}

	if err := settings.Parse(); err != nil {
		log.Fatalf("failed to parse settings: %v", err)
	}

	if err := tracegen.Process(settings, flags.Args(), update, getResolver); err != nil {
		log.Fatalf("failed to process: %v", err)
	}
}
