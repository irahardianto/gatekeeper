// Package main is the entry point for the gatekeeper CLI binary.
package main

import (
	"os"

	"github.com/irahardianto/gatekeeper/cmd/gatekeeper/commands"
)

func main() {
	if err := commands.Execute(); err != nil {
		os.Exit(1)
	}
}
