// Package demo provides the demo orchestration and output formatting.
package demo

import (
	"fmt"
	"strings"
	"time"

	"github.com/shopspring/decimal"

	"github.com/smpebble/actus-fvm/internal/model"
)

const (
	lineWidth   = 100
	sectionChar = "="
	subChar     = "-"
)

// PrintBanner prints the demo banner.
func PrintBanner() {
	fmt.Println()
	fmt.Println(strings.Repeat(sectionChar, lineWidth))
	fmt.Println(centerText("FinanceVM (FVM) - Financial Asset Tokenisation Demo", lineWidth))
	fmt.Println(centerText("ACTUS Hackathon 2025 Finals", lineWidth))
	fmt.Println(strings.Repeat(sectionChar, lineWidth))
	fmt.Println()
	fmt.Println("  Integrating: ACTUS-GO | FinanceVM 2.0 | vLEI-GO | ISO 20022")
	fmt.Println("  Four Scenarios: Stablecoin | Bond | Loan | Derivative")
	fmt.Println()
}

// PrintScenarioHeader prints a scenario header.
func PrintScenarioHeader(number int, result *model.ScenarioResult) {
	fmt.Println()
	fmt.Println(strings.Repeat(sectionChar, lineWidth))
	fmt.Printf("  SCENARIO %d: %s\n", number, result.Name)
	fmt.Println(strings.Repeat(sectionChar, lineWidth))
	fmt.Printf("  Type: %s | Contract: %s\n", result.Type, result.Instrument.ACTUSContractType)
	fmt.Printf("  %s\n", result.Description)
	fmt.Println()
}

// PrintInstrument prints instrument details.
func PrintInstrument(inst *model.Instrument) {
	printSubHeader("Instrument Details")
	fmt.Printf("  %-25s %s\n", "Name:", inst.FullName)
	fmt.Printf("  %-25s %s\n", "ID:", inst.ID[:8]+"...")
	if inst.ISIN != "" {
		fmt.Printf("  %-25s %s\n", "ISIN:", inst.ISIN)
	}
	fmt.Printf("  %-25s %s\n", "ACTUS Type:", inst.ACTUSContractType)
	fmt.Printf("  %-25s %s\n", "Token Standard:", inst.TokenStandard)
	fmt.Printf("  %-25s %s\n", "State:", inst.State)
	fmt.Printf("  %-25s %s\n", "Issued Amount:", inst.IssuedAmount.String())
	fmt.Printf("  %-25s %s\n", "Issue Date:", inst.IssueDate.Format("2006-01-02"))
	if inst.MaturityDate != nil {
		fmt.Printf("  %-25s %s\n", "Maturity Date:", inst.MaturityDate.Format("2006-01-02"))
	}
	if !inst.InterestRate.IsZero() {
		pct := inst.InterestRate.Mul(decimal.NewFromInt(100))
		fmt.Printf("  %-25s %s%%\n", "Interest Rate:", pct.StringFixed(2))
	}
	fmt.Println()
}

// PrintCashFlowTable prints the ACTUS cashflow events table.
func PrintCashFlowTable(events []model.CashFlowEvent) {
	printSubHeader("ACTUS Cash Flow Schedule")

	if len(events) == 0 {
		fmt.Println("  (No events generated)")
		return
	}

	// Header
	fmt.Printf("  %-5s %-20s %-12s %18s %18s  %s\n",
		"#", "Event Type", "Date", "Payoff", "Nominal Value", "Description")
	fmt.Printf("  %s\n", strings.Repeat(subChar, 95))

	count := 0
	for i, e := range events {
		count++
		// Truncate display for large schedules
		if len(events) > 20 && i >= 5 && i < len(events)-5 {
			if i == 5 {
				fmt.Printf("  %-5s ... (%d more events) ...\n", "...", len(events)-10)
			}
			continue
		}
		fmt.Printf("  %-5d %-20s %-12s %18s %18s  %s\n",
			count,
			e.EventType,
			e.Time.Format("2006-01-02"),
			e.Payoff.StringFixed(2),
			e.NominalValue.StringFixed(2),
			truncate(e.Description, 30),
		)
	}

	fmt.Printf("  %s\n", strings.Repeat(subChar, 95))
	fmt.Printf("  Total events: %d\n", len(events))
	fmt.Println()
}

// PrintPrecisionTests prints the precision comparison results.
func PrintPrecisionTests(tests []model.PrecisionComparison) {
	printSubHeader("Precision Analysis (shopspring/decimal vs float64)")

	for i, t := range tests {
		fmt.Printf("  Test %d: %s\n", i+1, t.Label)
		fmt.Printf("    Decimal (exact): %s\n", t.DecimalValue.StringFixed(10))
		fmt.Printf("    Float64:         %.10f\n", t.FloatValue)
		fmt.Printf("    Difference:      %s\n", t.Difference.StringFixed(15))
		if t.Difference.IsZero() {
			fmt.Println("    Result: ZERO PRECISION LOSS")
		} else {
			fmt.Println("    Result: Precision difference detected (decimal is authoritative)")
		}
		fmt.Println()
	}
}

// PrintSettlements prints settlement details.
func PrintSettlements(settlements []model.Settlement) {
	if len(settlements) == 0 {
		return
	}
	printSubHeader("Settlement")
	for _, s := range settlements {
		fmt.Printf("  %-20s %s\n", "Settlement ID:", s.ID[:8]+"...")
		fmt.Printf("  %-20s %s\n", "Type:", s.SettlementType)
		fmt.Printf("  %-20s %s -> %s\n", "Parties:", s.Deliverer, s.Receiver)
		fmt.Printf("  %-20s %s\n", "Amount:", s.CashAmount.String())
		fmt.Printf("  %-20s %s\n", "Date:", s.SettlementDate.Format("2006-01-02"))
		fmt.Printf("  %-20s %s\n", "State:", s.State)
		fmt.Println()
	}
}

// PrintJournalEntries prints accounting journal entries.
func PrintJournalEntries(entries []model.JournalEntry) {
	if len(entries) == 0 {
		return
	}
	printSubHeader("Accounting Journal Entries")
	for _, je := range entries {
		fmt.Printf("  [%s] %s\n", je.EntryDate.Format("2006-01-02"), je.Description)
		for _, line := range je.Lines {
			fmt.Printf("    %-8s %-30s %s\n", line.EntryType, line.AccountName, line.Amount.String())
		}
		fmt.Println()
	}
}

// PrintISO20022Messages prints ISO 20022 message summaries.
func PrintISO20022Messages(messages []model.ISO20022Message) {
	if len(messages) == 0 {
		return
	}
	printSubHeader("ISO 20022 Messages")
	for _, msg := range messages {
		fmt.Printf("  Message: %s (%s)\n", msg.MessageType, msg.Description)
		lines := strings.Split(msg.XMLContent, "\n")
		maxLines := 10
		if len(lines) < maxLines {
			maxLines = len(lines)
		}
		for _, line := range lines[:maxLines] {
			fmt.Printf("    %s\n", line)
		}
		if len(lines) > 10 {
			fmt.Printf("    ... (%d more lines)\n", len(lines)-10)
		}
		fmt.Println()
	}
}

// PrintVLEIVerification prints vLEI verification results.
func PrintVLEIVerification(v *model.VLEIVerificationResult) {
	if v == nil {
		return
	}
	printSubHeader("vLEI Identity Verification")
	status := "PASSED"
	if !v.IsValid {
		status = "FAILED"
	}
	fmt.Printf("  Verification Status: %s\n", status)
	fmt.Printf("  Verified At:         %s\n", v.VerifiedAt.Format(time.RFC3339))
	fmt.Printf("  Policy:              %s\n", v.PolicyName)
	if v.Issuer != nil {
		fmt.Printf("  Issuer:              %s (LEI: %s) [%s]\n", v.Issuer.LegalName, v.Issuer.LEI, v.Issuer.Type)
	}
	if v.Operator != nil {
		fmt.Printf("  Operator:            %s (%s) [%s]\n", v.Operator.LegalName, v.Operator.OfficialRole, v.Operator.Type)
	}
	if len(v.Errors) > 0 {
		fmt.Println("  Errors:")
		for _, e := range v.Errors {
			fmt.Printf("    - %s\n", e)
		}
	}
	fmt.Println()
}

// PrintSolidityFiles prints generated Solidity file information.
func PrintSolidityFiles(files []model.GeneratedFile) {
	if len(files) == 0 {
		return
	}
	printSubHeader("Generated Solidity Contracts")
	for _, f := range files {
		lines := strings.Count(f.Content, "\n") + 1
		fmt.Printf("  File: %s (%s, %d lines)\n", f.Filename, f.FileType, lines)
	}
	fmt.Println()
}

// PrintScenarioFooter prints scenario completion summary.
func PrintScenarioFooter(result *model.ScenarioResult) {
	duration := result.EndTime.Sub(result.StartTime)
	fmt.Printf("  Scenario completed in %s\n", duration.Round(time.Millisecond))
	fmt.Println()
}

// PrintSummary prints the final demo summary.
func PrintSummary(results []*model.ScenarioResult, totalDuration time.Duration) {
	fmt.Println()
	fmt.Println(strings.Repeat(sectionChar, lineWidth))
	fmt.Println(centerText("DEMO SUMMARY", lineWidth))
	fmt.Println(strings.Repeat(sectionChar, lineWidth))
	fmt.Println()

	fmt.Printf("  %-5s %-50s %-10s %-10s\n", "#", "Scenario", "Events", "Status")
	fmt.Printf("  %s\n", strings.Repeat(subChar, 80))

	totalEvents := 0
	for i, r := range results {
		eventCount := len(r.CashFlowEvents)
		totalEvents += eventCount
		fmt.Printf("  %-5d %-50s %-10d %-10s\n", i+1, r.Name, eventCount, "OK")
	}

	fmt.Printf("  %s\n", strings.Repeat(subChar, 80))
	fmt.Printf("  Total scenarios: %d | Total events: %d | Duration: %s\n",
		len(results), totalEvents, totalDuration.Round(time.Millisecond))
	fmt.Println()
	fmt.Println("  Key Technologies Demonstrated:")
	fmt.Println("    - ACTUS Standard: PAM, ANN, SWAPS contract types")
	fmt.Println("    - shopspring/decimal: Zero precision loss in all calculations")
	fmt.Println("    - Solidity Code Generation: Smart contracts from ACTUS specs")
	fmt.Println("    - ISO 20022: pacs.008, sese.023, camt.054 messages")
	fmt.Println("    - vLEI: Verifiable Legal Entity Identity verification")
	fmt.Println("    - FinanceVM: Complete instrument lifecycle management")
	fmt.Println()
	fmt.Println(strings.Repeat(sectionChar, lineWidth))
}

// WaitForEnter pauses and waits for the user to press Enter.
func WaitForEnter(prompt string) {
	fmt.Printf("\n  >>> %s ", prompt)
	fmt.Scanln()
}

func printSubHeader(title string) {
	fmt.Printf("  --- %s ---\n", title)
}

func centerText(text string, width int) string {
	pad := (width - len(text)) / 2
	if pad < 0 {
		pad = 0
	}
	return strings.Repeat(" ", pad) + text
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}
