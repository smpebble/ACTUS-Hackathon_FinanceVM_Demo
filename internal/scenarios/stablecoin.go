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

// StablecoinScenario demonstrates a 1:1 fiat-backed stablecoin (TWDX).
type StablecoinScenario struct{}

func (s *StablecoinScenario) Name() string             { return "Stablecoin (TWDX) Minting & Redemption" }
func (s *StablecoinScenario) Type() model.ScenarioType { return model.ScenarioStablecoin }

func (s *StablecoinScenario) Run(ctx *ScenarioContext, params map[string]interface{}) (*model.ScenarioResult, error) {
	result := &model.ScenarioResult{
		Type:        model.ScenarioStablecoin,
		Name:        s.Name(),
		Description: "1:1 TWD-backed stablecoin. Mint 1,000,000 TWDX with zero precision loss.",
		StartTime:   time.Now(),
	}

	// Create ACTUS PAM contract for stablecoin (zero interest, perpetual-like)
	now := time.Now()
	issueDate := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	// Use a 30-year maturity to simulate long-duration stablecoin backing
	maturityDate := issueDate.AddDate(30, 0, 0)

	notionalVal := getParamFloat(params, "notional", 1000000.0)
	notional := decimal.NewFromFloat(notionalVal)

	attrs := &types.ContractAttributes{
		ContractID:          "TWDX-STABLECOIN-001",
		ContractType:        "PAM",
		ContractRole:        "RPA",
		Currency:            "TWD",
		DayCountConvention:  dayCountA360(),
		StatusDate:          issueDate.AddDate(0, 0, -1),
		InitialExchangeDate: issueDate,
		MaturityDate:        makeTimePtr(maturityDate),
		NotionalPrincipal:   notional,
		NominalInterestRate: decimal.Zero,
	}

	pamContract, schedule, state, err := ctx.ACTUS.CreatePAMContract(attrs)
	if err != nil {
		return nil, fmt.Errorf("stablecoin PAM contract creation failed: %w", err)
	}

	// Simulate cashflows (only IED and MD for zero-interest)
	rf := adapter.NewStaticRiskFactorObserver(nil)
	cashflows, err := ctx.ACTUS.SimulateCashflows(pamContract, pamContract, schedule, state, rf)
	if err != nil {
		return nil, fmt.Errorf("stablecoin cashflow simulation failed: %w", err)
	}
	result.CashFlowEvents = cashflows

	// Precision test: 1:1 mapping must be exact
	mintAmount := notional
	mintFloat := notionalVal
	result.PrecisionTests = append(result.PrecisionTests,
		comparePrecision(fmt.Sprintf("Stablecoin Mint: %.2f TWD -> TWDX", mintFloat), mintAmount, mintFloat),
	)

	// Redemption precision
	redeemAmount := decimal.NewFromFloat(999999.99)
	redeemFloat := 999999.99
	remainDecimal := mintAmount.Sub(redeemAmount)
	remainFloat := mintFloat - redeemFloat
	result.PrecisionTests = append(result.PrecisionTests,
		comparePrecision("Remaining after 999,999.99 redemption", remainDecimal, remainFloat),
	)

	// FVM instrument lifecycle
	inst := ctx.FVM.CreateInstrument(
		"TWDX Stablecoin",
		"1:1 TWD-backed stablecoin token",
		"AIMICHIA TECHNOLOGY CO., LTD.",
		model.CurrencyTWD,
		"PAM",
		model.TokenStandardERC20,
	)
	if err := ctx.FVM.IssueInstrument(inst.ID, mintAmount, issueDate); err != nil {
		return nil, err
	}
	if err := ctx.FVM.ActivateInstrument(inst.ID); err != nil {
		return nil, err
	}
	result.Instrument = inst

	// Generate Solidity ERC-20 contract
	outputDir := filepath.Join(ctx.BaseDir, "generated", "stablecoin")
	solidityFiles, err := ctx.Codegen.GenerateSolidityContract("TWDXStablecoin", "PAM", schedule, outputDir)
	if err != nil {
		return nil, fmt.Errorf("stablecoin solidity generation failed: %w", err)
	}
	result.SolidityFiles = solidityFiles

	// Settlement, accounting, ISO 20022, vLEI
	verifyAndSettle(ctx, result)

	// Attach Risk Analysis
	result.RiskAnalysis = CalculateRiskAnalysis(result.CashFlowEvents, "PAM", params)

	result.EndTime = time.Now()
	return result, nil
}
