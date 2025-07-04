// Package main is the entry point for the amux CLI application.
package main

import (
	"fmt"
	"os"

	"github.com/aki/amux/internal/cli/commands"
)

func main() {
	if err := commands.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
