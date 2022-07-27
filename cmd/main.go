package main

import (
	"github.com/jenkins-x-plugins/jx-semanticcheck/cmd/app"
	"os"
)

// Entrypoint for the command
func main() {
	if err := app.Run(nil); err != nil {
		os.Exit(1)
	}
	os.Exit(0)
}
