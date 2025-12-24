package main

import (
	"os"

	"github.com/tuannvm/pagent/internal/cmd"
)

// Version is set at build time via ldflags
var Version = "dev"

func main() {
	cmd.SetVersion(Version)
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}
