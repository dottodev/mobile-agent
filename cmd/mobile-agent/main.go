package main

import (
	"fmt"
	"os"

	"mobile-agent/internal/ui"
)

func main() {
	if err := ui.Run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "mobile-agent:", err)
		os.Exit(1)
	}
}
