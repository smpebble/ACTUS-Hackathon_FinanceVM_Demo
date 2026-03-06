// Package scenarios implements the 4 core financial demo scenarios.
package scenarios

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os/exec"
	"time"

	"github.com/shopspring/decimal"
	"github.com/yourusername/actus-go/pkg/actus/types"

	"github.com/smpebble/actus-fvm/internal/adapter"
	"github.com/smpebble/actus-fvm/internal/model"
)

// ScenarioContext holds shared dependencies for all scenarios.
type ScenarioContext struct {
	ACTUS   *adapter.ACTUSClient
	FVM     *adapter.FVMClient
	VLEI    *adapter.VLEIClient
	ISO     *adapter.ISO20022Client
	Codegen *adapter.CodegenClient
	BaseDir string // Base directory for generated files
}

// NewScenarioContext creates a new context with all adapters initialized.
func NewScenarioContext(baseDir string) *ScenarioContext {
	return &ScenarioContext{
		ACTUS:   adapter.NewACTUSClient(),
		FVM:     adapter.NewFVMClient(),
		VLEI:    adapter.NewVLEIClient(),
		ISO:     adapter.NewISO20022Client("FVMBTWTPXXX", "FinanceVM Taiwan"),
		Codegen: adapter.NewCodegenClient(),
		BaseDir: baseDir,
	}
}

// Scenario is the interface all scenarios implement.
type Scenario interface {
	Name() string
	Type() model.ScenarioType
	Run(ctx *ScenarioContext, params map[string]interface{}) (*model.ScenarioResult, error)
}

// verifyAndSettle performs the common vLEI verification + settlement + accounting + ISO 20022 flow.
func verifyAndSettle(ctx *ScenarioContext, result *model.ScenarioResult) {
	// vLEI verification
	issuerCred := ctx.VLEI.CreateIssuerCredential(
		"9845008A70A0AA114522",
		"AIMICHIA TECHNOLOGY CO., LTD.",
		"TW",
	)
	operatorCred := ctx.VLEI.CreateOperatorCredential(
		"9845008A70A0AA114522",
		"Demo Operator",
		"Chief Financial Officer",
	)
	result.VLEIVerification = ctx.VLEI.VerifyTransaction(issuerCred, operatorCred)

	// Create settlement for the initial exchange
	if result.Instrument != nil && len(result.CashFlowEvents) > 0 {
		settlement := ctx.FVM.CreateSettlement(
			result.Instrument.ID,
			model.SettlementTypeDVP,
			"AIMICHIA TECHNOLOGY CO., LTD.",
			"Investor A",
			result.Instrument.IssuedAmount,
			result.Instrument.IssueDate,
		)
		ctx.FVM.SettlePayment(settlement)
		result.Settlements = append(result.Settlements, *settlement)

		// Journal entry
		je := ctx.FVM.CreateJournalEntry(
			result.Instrument.IssueDate,
			"Initial exchange - "+result.Name,
			"Cash Account",
			"Securities Account",
			result.Instrument.IssuedAmount,
		)
		result.JournalEntries = append(result.JournalEntries, *je)

		// ISO 20022 messages
		pacs008 := ctx.ISO.GeneratePacs008(settlement)
		result.ISO20022Messages = append(result.ISO20022Messages, pacs008)

		if result.Instrument.ISIN != "" {
			sese023 := ctx.ISO.GenerateSese023(settlement, result.Instrument.ISIN)
			result.ISO20022Messages = append(result.ISO20022Messages, sese023)
		}
	}
}

// comparePrecision computes a precision comparison between decimal and float64.
func comparePrecision(label string, decimalResult decimal.Decimal, floatResult float64) model.PrecisionComparison {
	floatAsDecimal := decimal.NewFromFloat(floatResult)
	diff := decimalResult.Sub(floatAsDecimal).Abs()
	return model.PrecisionComparison{
		Label:        label,
		DecimalValue: decimalResult,
		FloatValue:   floatResult,
		Difference:   diff,
	}
}

// makeTimePtr creates a pointer to a time.Time value.
func makeTimePtr(t time.Time) *time.Time {
	return &t
}

// Helper to create day count convention
func dayCountA360() types.DayCountConvention {
	return "A/360"
}

// yearFractionAct360Decimal calculates the year fraction using Act/360 with decimal precision.
func yearFractionAct360Decimal(start, end time.Time) decimal.Decimal {
	days := decimal.NewFromFloat(end.Sub(start).Hours() / 24)
	return days.Div(decimal.NewFromInt(360))
}

// getParamFloat safely extracts a float parameter with a fallback default.
func getParamFloat(params map[string]interface{}, key string, def float64) float64 {
	if params == nil {
		return def
	}
	if v, ok := params[key]; ok {
		switch val := v.(type) {
		case float64:
			return val
		case float32:
			return float64(val)
		case int:
			return float64(val)
		case int64:
			return float64(val)
		}
	}
	return def
}

// CalculateRiskAnalysis generates risk metrics based on contract type and cash flows.
func CalculateRiskAnalysis(events []model.CashFlowEvent, contractType string, params map[string]interface{}) *model.RiskAnalysisResult {
	// Prepare JSON payload for the Python script
	payload := map[string]interface{}{
		"contractType": contractType,
		"notional":     params["notional"],
		"rate":         params["rate"],
		"events":       events,
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		fmt.Printf("Error marshaling risk payload: %v\n", err)
		return fallbackRiskResult()
	}

	cmd := exec.Command("python", "scripts/quantlib_risk.py")
	// cmd.Dir will be inherited as the project root
	cmd.Stdin = bytes.NewReader(payloadBytes)

	var out bytes.Buffer
	cmd.Stdout = &out
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		fmt.Printf("Python QuantLib execution error: %v, stderr: %s\n", err, stderr.String())
		return fallbackRiskResult()
	}

	var ptResult struct {
		MarketRisk       string `json:"marketRisk"`
		CreditRisk       string `json:"creditRisk"`
		LiquidityRisk    string `json:"liquidityRisk"`
		CounterpartyRisk string `json:"counterpartyRisk"`
		StressTestImpact string `json:"stressTestImpact"`
		Engine           string `json:"engine"`
	}

	if err := json.Unmarshal(out.Bytes(), &ptResult); err != nil {
		fmt.Printf("Error unmarshaling Python risk result: %v, body: %s\n", err, out.String())
		return fallbackRiskResult()
	}

	mr, _ := decimal.NewFromString(ptResult.MarketRisk)
	cr, _ := decimal.NewFromString(ptResult.CreditRisk)
	lr, _ := decimal.NewFromString(ptResult.LiquidityRisk)
	cp, _ := decimal.NewFromString(ptResult.CounterpartyRisk)
	st, _ := decimal.NewFromString(ptResult.StressTestImpact)

	return &model.RiskAnalysisResult{
		MarketRisk:       mr,
		CreditRisk:       cr,
		LiquidityRisk:    lr,
		CounterpartyRisk: cp,
		StressTestImpact: st,
	}
}

func fallbackRiskResult() *model.RiskAnalysisResult {
	return &model.RiskAnalysisResult{
		MarketRisk:       decimal.NewFromFloat(100),
		CreditRisk:       decimal.NewFromFloat(50),
		LiquidityRisk:    decimal.NewFromFloat(20),
		CounterpartyRisk: decimal.NewFromFloat(10),
		StressTestImpact: decimal.NewFromFloat(-500),
	}
}
