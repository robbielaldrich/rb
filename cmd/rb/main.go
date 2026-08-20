package main

import (
	"fmt"
	"log"
	"os"

	"rb/cli"
)

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}

	cmd, args := os.Args[1], os.Args[2:]

	var err error
	switch cmd {
	case "download-cards":
		err = cli.RunDownloadCards(args)
	case "collection":
		err = cli.RunCollection(args)
	case "-h", "-help", "--help", "help":
		usage()
	default:
		fmt.Fprintf(os.Stderr, "rb: unknown command %q\n\n", cmd)
		usage()
		os.Exit(2)
	}

	// Every command returns its error rather than exiting, so this is the one
	// place the program dies and the message carries the whole wrapped chain.
	if err != nil {
		log.Fatalf("rb %s: %v", cmd, err)
	}
}

func usage() {
	fmt.Fprint(os.Stderr, `usage: rb <command> [args]

commands:
  download-cards   download card and set data from the Riftcodex API
  collection       add cards to your collection, interactively
`)
}
