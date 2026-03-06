package scenarios

import (
	"fmt"
	"path/filepath"
	"time"

	"github.com/shopspring/decimal"
	"github.com/yourusername/actus-go/pkg/actus/types"

	"github.com/smpebble/actus-fvm/internal/adapter"
	"github.com/smpebble/actus-fvm/internal/model"
)

// DerivativeScenario demonstrates a 3-year interest rate swap (IRS).
type DerivativeScenario struct{}

func (s *DerivativeScenario) Name() string {
	return "Interest Rate Swap (3Y, Fixed 3.0% vs TAIBOR+50bps)"
}
func (s *DerivativeScenario) Type() model.ScenarioType { return model.ScenarioDerivative }

func (s *DerivativeScenario) Run(ctx *ScenarioContext, params map[string]interface{}) (*model.ScenarioResult, error) {
	result := &model.ScenarioResult{
		Type:        model.ScenarioDerivative,
		Name:        s.Name(),
		Description: "3-year IRS, 50M TWD notional, fixed 3.0% vs floating TAIBOR+50bps, quarterly payments.",
		StartTime:   time.Now(),
	}

	// Swap parameters
	issueDate := time.Date(2025, 4, 1, 0, 0, 0, 0, time.UTC)
	maturityDate := time.Date(2028, 4, 1, 0, 0, 0, 0, time.UTC)
	notionalVal := getParamFloat(params, "notional", 50000000.0) // 50M TWD
	notional := decimal.NewFromFloat(notionalVal)
	rateVal := getParamFloat(params, "rate", 0.03) // 3.0%
	fixedRate := decimal.NewFromFloat(rateVal)
	spreadVal := getParamFloat(params, "spread", 0.005) // 50 bps
	spread := decimal.NewFromFloat(spreadVal)

	// For SWAPS, we simulate using PAM for the fixed leg as a simplified demo
	// (The actual SWAPS contract may need additional setup for two legs)
	attrs := &types.ContractAttributes{
		ContractID:                       "IRS-TW-2025-001",
		ContractType:                     "PAM",
		ContractRole:                     "RFL", // Receive Fixed Leg
		Currency:                         "TWD",
		DayCountConvention:               dayCountA360(),
		StatusDate:                       issueDate.AddDate(0, 0, -1),
		InitialExchangeDate:              issueDate,
		MaturityDate:                     makeTimePtr(maturityDate),
		NotionalPrincipal:                notional,
		NominalInterestRate:              fixedRate,
		CycleOfInterestPayment:           "P3M",
		CycleAnchorDateOfInterestPayment: makeTimePtr(time.Date(2025, 7, 1, 0, 0, 0, 0, time.UTC)),
	}

	contract, schedule, state, err := ctx.ACTUS.CreatePAMContract(attrs)
	if err != nil {
		return nil, fmt.Errorf("derivative contract creation failed: %w", err)
	}

	// Set up floating rate observer (TAIBOR simulated rates)
	taiborRates := map[string]decimal.Decimal{
		"TAIBOR3M": decimal.NewFromFloat(0.025), // 2.5% base TAIBOR
	}
	rf := adapter.NewStaticRiskFactorObserver(taiborRates)

	cashflows, err := ctx.ACTUS.SimulateCashflows(contract, contract, schedule, state, rf)
	if err != nil {
		return nil, fmt.Errorf("derivative cashflow simulation failed: %w", err)
	}

	// Generate both fixed and floating leg cashflows for display
	var enrichedCashflows []model.CashFlowEvent

	// TAIBOR rate for floating leg simulation
	taiborBase := decimal.NewFromFloat(0.025)
	floatingRate := taiborBase.Add(spread) // 2.5% + 0.5% = 3.0%

	for _, cf := range cashflows {
		// Fixed leg event
		enrichedCashflows = append(enrichedCashflows, model.CashFlowEvent{
			EventType:    cf.EventType + " (Fixed Leg)",
			Time:         cf.Time,
			Payoff:       cf.Payoff,
			Currency:     cf.Currency,
			NominalValue: cf.NominalValue,
			Description:  fmt.Sprintf("Fixed leg payment @ %.2f%%", fixedRate.Mul(decimal.NewFromInt(100)).InexactFloat64()),
		})

		// Simulate floating leg for IP events
		if cf.EventType == "IP" {
			// Calculate floating leg payment for the quarter
			quarterFraction := yearFractionAct360Decimal(cf.Time.AddDate(0, -3, 0), cf.Time)
			floatingPayoff := notional.Mul(floatingRate).Mul(quarterFraction).Neg()

			enrichedCashflows = append(enrichedCashflows, model.CashFlowEvent{
				EventType:    "IP (Floating Leg)",
				Time:         cf.Time,
				Payoff:       floatingPayoff,
				Currency:     cf.Currency,
				NominalValue: cf.NominalValue,
				Description:  fmt.Sprintf("Floating leg payment @ TAIBOR(%.2f%%) + 50bps = %.2f%%", taiborBase.Mul(decimal.NewFromInt(100)).InexactFloat64(), floatingRate.Mul(decimal.NewFromInt(100)).InexactFloat64()),
			})

			// Net settlement
			netPayoff := cf.Payoff.Add(floatingPayoff)
			direction := "Receive"
			if netPayoff.IsNegative() {
				direction = "Pay"
			}
			enrichedCashflows = append(enrichedCashflows, model.CashFlowEvent{
				EventType:    "NET",
				Time:         cf.Time,
				Payoff:       netPayoff,
				Currency:     cf.Currency,
				NominalValue: cf.NominalValue,
				Description:  fmt.Sprintf("Net settlement (%s)", direction),
			})
		}
	}
	result.CashFlowEvents = enrichedCashflows

	// Precision tests
	// Fixed leg quarterly payment: 50M * 3.0% * (91/360)
	days91 := decimal.NewFromFloat(91)
	decFixedPayment := notional.Mul(fixedRate).Mul(days91.Div(decimal.NewFromInt(360)))
	floatFixedPayment := notionalVal * rateVal * (91.0 / 360.0)

	result.PrecisionTests = append(result.PrecisionTests,
		comparePrecision("Fixed leg quarterly payment (91 days)", decFixedPayment, floatFixedPayment),
	)

	// Floating leg quarterly payment: 50M * 3.0% * (91/360)
	floatFloatingRate := 0.025 + spreadVal
	decFloatingPayment := notional.Mul(floatingRate).Mul(days91.Div(decimal.NewFromInt(360)))
	floatFloatingPayment := notionalVal * floatFloatingRate * (91.0 / 360.0)

	result.PrecisionTests = append(result.PrecisionTests,
		comparePrecision("Floating leg quarterly payment (91 days)", decFloatingPayment, floatFloatingPayment),
	)

	// Net payment
	decNet := decFixedPayment.Sub(decFloatingPayment)
	floatNet := floatFixedPayment - floatFloatingPayment

	result.PrecisionTests = append(result.PrecisionTests,
		comparePrecision("Net settlement amount", decNet, floatNet),
	)

	// FVM instrument lifecycle
	inst := ctx.FVM.CreateInstrument(
		"IRS TWD 3Y Fixed-Float",
		"3-year interest rate swap, fixed 3.0% vs TAIBOR+50bps",
		"AIMICHIA TECHNOLOGY CO., LTD.",
		model.CurrencyTWD,
		"SWAPS",
		model.TokenStandardCustom,
	)
	if err := ctx.FVM.IssueInstrument(inst.ID, notional, issueDate); err != nil {
		return nil, err
	}
	if err := ctx.FVM.ActivateInstrument(inst.ID); err != nil {
		return nil, err
	}
	inst.MaturityDate = makeTimePtr(maturityDate)
	inst.InterestRate = fixedRate
	result.Instrument = inst

	// Generate Solidity Swap contract
	outputDir := filepath.Join(ctx.BaseDir, "generated", "derivative")
	solidityFiles, err := ctx.Codegen.GenerateSolidityContract("InterestRateSwap", "SWAPS", schedule, outputDir)
	if err != nil {
		return nil, fmt.Errorf("derivative solidity generation failed: %w", err)
	}
	result.SolidityFiles = solidityFiles

	// Settlement, accounting, ISO 20022, vLEI
	verifyAndSettle(ctx, result)

	// Attach Risk Analysis
	result.RiskAnalysis = CalculateRiskAnalysis(result.CashFlowEvents, "SWAPS", params)

	result.EndTime = time.Now()
	return result, nil
}
