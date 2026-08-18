// Command rb is the Riftbound collection tool. It dispatches on its first
// positional argument to one of the subcommands below.
package main

import (
	"fmt"
	"os"
)

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}

	cmd, args := os.Args[1], os.Args[2:]
	switch cmd {
	case "download-cards":
		runDownloadCards(args)
	case "collection":
		runCollection(args)
	case "-h", "-help", "--help", "help":
		usage()
	default:
		fmt.Fprintf(os.Stderr, "rb: unknown command %q\n\n", cmd)
		usage()
		os.Exit(2)
	}
}

func usage() {
	fmt.Fprint(os.Stderr, `usage: rb <command> [args]

commands:
  download-cards   download card and set data from the Riftcodex API
  collection       browse your card collection
`)
}
