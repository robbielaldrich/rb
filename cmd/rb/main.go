package main

import (
	"fmt"
	"log"
	"os"

	"rb/cli"
)

func main() {
	err := cli.Run(os.Args[1:])
	switch {
	case cli.IsUsage(err):
		fmt.Fprintf(os.Stderr, "rb: %v\n\n", err)
		cli.Usage(os.Stderr)
		os.Exit(2)
	case err != nil:
		log.Fatalf("rb: %v", err)
	}
}
