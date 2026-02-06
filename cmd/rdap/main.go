package main

import (
	"os"

	rdap "github.com/DataPulse/openrdap"
)

func main() {
	exitCode := rdap.RunCLI(os.Args[1:], os.Stdout, os.Stderr, rdap.CLIOptions{})

	os.Exit(exitCode)
}
