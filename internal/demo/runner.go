package demo

import (
	"fmt"
	"time"

	"github.com/smpebble/actus-fvm/internal/model"
	"github.com/smpebble/actus-fvm/internal/scenarios"
)

// Runner orchestrates all demo scenarios.
type Runner struct {
	ctx       *scenarios.ScenarioContext
	scenarios []scenarios.Scenario
}

// NewRunner creates a new demo runner with all 4 scenarios.
func NewRunner(baseDir string) *Runner {
	return &Runner{
		ctx: scenarios.NewScenarioContext(baseDir),
		scenarios: []scenarios.Scenario{
			&scenarios.StablecoinScenario{},
			&scenarios.BondScenario{},
			&scenarios.LoanScenario{},
			&scenarios.DerivativeScenario{},
		},
	}
}

// Run executes all scenarios one stage at a time, pausing between each.
func (r *Runner) Run() error {
	PrintBanner()
	WaitForEnter(fmt.Sprintf("Press Enter to start Scenario 1/%d...", len(r.scenarios)))

	startTime := time.Now()
	var results []*model.ScenarioResult

	for i, scenario := range r.scenarios {
		fmt.Printf("\n  Running scenario %d/%d: %s ...\n", i+1, len(r.scenarios), scenario.Name())

		result, err := scenario.Run(r.ctx, nil)
		if err != nil {
			fmt.Printf("\n  ERROR in scenario %d (%s): %v\n", i+1, scenario.Name(), err)
			fmt.Println("  Continuing with next scenario...")
			continue
		}

		results = append(results, result)
		printScenarioResult(i+1, result)

		if i < len(r.scenarios)-1 {
			WaitForEnter(fmt.Sprintf("Press Enter to continue to Scenario %d/%d: %s...", i+2, len(r.scenarios), r.scenarios[i+1].Name()))
		}
	}

	WaitForEnter("Press Enter to view the final summary...")
	totalDuration := time.Since(startTime)
	PrintSummary(results, totalDuration)

	return nil
}

// RunSingle runs a single scenario by type.
func (r *Runner) RunSingle(scenarioType model.ScenarioType) error {
	for _, scenario := range r.scenarios {
		if scenario.Type() == scenarioType {
			result, err := scenario.Run(r.ctx, nil)
			if err != nil {
				return err
			}
			printScenarioResult(1, result)
			return nil
		}
	}
	return fmt.Errorf("scenario not found: %s", scenarioType)
}

// RunAllAndCollect executes all scenarios and returns results without printing.
func (r *Runner) RunAllAndCollect(params map[string]interface{}) ([]*model.ScenarioResult, error) {
	var results []*model.ScenarioResult
	for _, scenario := range r.scenarios {
		result, err := scenario.Run(r.ctx, params)
		if err != nil {
			return results, fmt.Errorf("scenario %s failed: %w", scenario.Name(), err)
		}
		results = append(results, result)
	}
	return results, nil
}

// RunScenario runs a single scenario by type and returns the result without printing.
func (r *Runner) RunScenario(scenarioType model.ScenarioType, params map[string]interface{}) (*model.ScenarioResult, error) {
	for _, scenario := range r.scenarios {
		if scenario.Type() == scenarioType {
			return scenario.Run(r.ctx, params)
		}
	}
	return nil, fmt.Errorf("scenario not found: %s", scenarioType)
}

func printScenarioResult(number int, result *model.ScenarioResult) {
	PrintScenarioHeader(number, result)

	PrintInstrument(result.Instrument)
	WaitForEnter("Press Enter to view Cash Flow Schedule...")

	PrintCashFlowTable(result.CashFlowEvents)
	WaitForEnter("Press Enter to view Precision Analysis...")

	PrintPrecisionTests(result.PrecisionTests)
	WaitForEnter("Press Enter to view Settlement & Accounting...")

	PrintSettlements(result.Settlements)
	PrintJournalEntries(result.JournalEntries)
	WaitForEnter("Press Enter to view ISO 20022 Messages...")

	PrintISO20022Messages(result.ISO20022Messages)
	WaitForEnter("Press Enter to view vLEI Verification...")

	PrintVLEIVerification(result.VLEIVerification)
	WaitForEnter("Press Enter to view Generated Solidity Contracts...")

	PrintSolidityFiles(result.SolidityFiles)
	PrintScenarioFooter(result)
}
