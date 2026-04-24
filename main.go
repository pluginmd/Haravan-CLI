// Package main is the entry point for haravan-cli.
package main

import (
	"fmt"
	"os"

	"github.com/pluginmd/haravan-cli/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
