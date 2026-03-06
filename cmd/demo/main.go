package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/smpebble/actus-fvm/internal/demo"
	"github.com/smpebble/actus-fvm/internal/model"
)

func main() {
	scenario := flag.String("scenario", "all", "Scenario to run: all, stablecoin, bond, loan, derivative")
	flag.Parse()

	baseDir, err := os.Getwd()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to get working directory: %v\n", err)
		os.Exit(1)
	}

	runner := demo.NewRunner(baseDir)

	switch strings.ToLower(*scenario) {
	case "all":
		if err := runner.Run(); err != nil {
			fmt.Fprintf(os.Stderr, "Demo failed: %v\n", err)
			os.Exit(1)
		}
	case "stablecoin":
		if err := runner.RunSingle(model.ScenarioStablecoin); err != nil {
			fmt.Fprintf(os.Stderr, "Scenario failed: %v\n", err)
			os.Exit(1)
		}
	case "bond":
		if err := runner.RunSingle(model.ScenarioBond); err != nil {
			fmt.Fprintf(os.Stderr, "Scenario failed: %v\n", err)
			os.Exit(1)
		}
	case "loan":
		if err := runner.RunSingle(model.ScenarioLoan); err != nil {
			fmt.Fprintf(os.Stderr, "Scenario failed: %v\n", err)
			os.Exit(1)
		}
	case "derivative":
		if err := runner.RunSingle(model.ScenarioDerivative); err != nil {
			fmt.Fprintf(os.Stderr, "Scenario failed: %v\n", err)
			os.Exit(1)
		}
	default:
		fmt.Fprintf(os.Stderr, "Unknown scenario: %s\n", *scenario)
		fmt.Fprintln(os.Stderr, "Available: all, stablecoin, bond, loan, derivative")
		os.Exit(1)
	}
}
