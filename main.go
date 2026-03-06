package main

import (
	"fmt"
	"os"

	"github.com/smpebble/actus-fvm/internal/demo"
)

func main() {
	baseDir, err := os.Getwd()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to get working directory: %v\n", err)
		os.Exit(1)
	}

	runner := demo.NewRunner(baseDir)
	if err := runner.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Demo failed: %v\n", err)
		os.Exit(1)
	}
}
