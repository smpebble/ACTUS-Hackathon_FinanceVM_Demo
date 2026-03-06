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

// LoanScenario demonstrates an SME loan with monthly amortization (ANN contract).
type LoanScenario struct{}

func (s *LoanScenario) Name() string             { return "SME Loan (5Y, 4.5%, Monthly Amortization)" }
func (s *LoanScenario) Type() model.ScenarioType { return model.ScenarioLoan }

func (s *LoanScenario) Run(ctx *ScenarioContext, params map[string]interface{}) (*model.ScenarioResult, error) {
	result := &model.ScenarioResult{
		Type:        model.ScenarioLoan,
		Name:        s.Name(),
		Description: "SME loan, 10M TWD, 5 years, 4.5% fixed rate, monthly payments. Full amortization schedule.",
		StartTime:   time.Now(),
	}

	// Loan parameters
	issueDate := time.Date(2025, 3, 1, 0, 0, 0, 0, time.UTC)
	maturityDate := time.Date(2030, 3, 1, 0, 0, 0, 0, time.UTC)

	notionalVal := getParamFloat(params, "notional", 10000000.0) // 10M TWD
	notional := decimal.NewFromFloat(notionalVal)
	rateVal := getParamFloat(params, "rate", 0.045) // 4.5%
	rate := decimal.NewFromFloat(rateVal)

	attrs := &types.ContractAttributes{
		ContractID:                           "LOAN-SME-2025-001",
		ContractType:                         "ANN",
		ContractRole:                         "RPA",
		Currency:                             "TWD",
		DayCountConvention:                   dayCountA360(),
		StatusDate:                           issueDate.AddDate(0, 0, -1),
		InitialExchangeDate:                  issueDate,
		MaturityDate:                         makeTimePtr(maturityDate),
		NotionalPrincipal:                    notional,
		NominalInterestRate:                  rate,
		CycleOfInterestPayment:               "P1M",
		CycleAnchorDateOfInterestPayment:     makeTimePtr(time.Date(2025, 4, 1, 0, 0, 0, 0, time.UTC)),
		CycleOfPrincipalRedemption:           "P1M",
		CycleAnchorDateOfPrincipalRedemption: makeTimePtr(time.Date(2025, 4, 1, 0, 0, 0, 0, time.UTC)),
	}

	contract, schedule, state, err := ctx.ACTUS.CreateANNContract(attrs)
	if err != nil {
		return nil, fmt.Errorf("loan ANN contract creation failed: %w", err)
	}

	// Simulate cashflows
	rf := adapter.NewStaticRiskFactorObserver(nil)
	cashflows, err := ctx.ACTUS.SimulateCashflows(contract, contract, schedule, state, rf)
	if err != nil {
		return nil, fmt.Errorf("loan cashflow simulation failed: %w", err)
	}
	result.CashFlowEvents = cashflows

	// Precision tests
	// Monthly payment calculation using annuity formula
	monthlyRate := rateVal / 12.0
	nPayments := 60.0
	// PMT = PV * r / (1 - (1+r)^-n)
	floatPMT := notionalVal * monthlyRate / (1 - 1/pow(1+monthlyRate, nPayments))

	decRate := rate.Div(decimal.NewFromInt(12))
	decOne := decimal.NewFromInt(1)
	// (1+r)^n
	onePlusR := decOne.Add(decRate)
	powN := powDecimal(onePlusR, 60)
	decPMT := notional.Mul(decRate).Mul(powN).Div(powN.Sub(decOne))

	result.PrecisionTests = append(result.PrecisionTests,
		comparePrecision("Monthly annuity payment (PMT formula)", decPMT, floatPMT),
	)

	// Total interest paid
	totalFloatPayments := floatPMT * 60
	totalFloatInterest := totalFloatPayments - notionalVal
	totalDecPayments := decPMT.Mul(decimal.NewFromInt(60))
	totalDecInterest := totalDecPayments.Sub(notional)

	result.PrecisionTests = append(result.PrecisionTests,
		comparePrecision("Total interest over loan lifetime", totalDecInterest, totalFloatInterest),
	)

	// FVM instrument lifecycle
	inst := ctx.FVM.CreateInstrument(
		"SME Loan 2025-001",
		"5-year SME loan with monthly amortization",
		"AIMICHIA TECHNOLOGY CO., LTD.",
		model.CurrencyTWD,
		"ANN",
		model.TokenStandardCustom,
	)
	if err := ctx.FVM.IssueInstrument(inst.ID, notional, issueDate); err != nil {
		return nil, err
	}
	if err := ctx.FVM.ActivateInstrument(inst.ID); err != nil {
		return nil, err
	}
	inst.MaturityDate = makeTimePtr(maturityDate)
	inst.InterestRate = rate
	result.Instrument = inst

	// Generate Solidity Loan contract
	outputDir := filepath.Join(ctx.BaseDir, "generated", "loan")
	solidityFiles, err := ctx.Codegen.GenerateSolidityContract("SMELoanContract", "ANN", schedule, outputDir)
	if err != nil {
		return nil, fmt.Errorf("loan solidity generation failed: %w", err)
	}
	result.SolidityFiles = solidityFiles

	// Settlement, accounting, ISO 20022, vLEI
	verifyAndSettle(ctx, result)

	// Attach Risk Analysis
	result.RiskAnalysis = CalculateRiskAnalysis(result.CashFlowEvents, "ANN", params)

	result.EndTime = time.Now()
	return result, nil
}

// pow computes base^exp for float64.
func pow(base, exp float64) float64 {
	result := 1.0
	for i := 0; i < int(exp); i++ {
		result *= base
	}
	return result
}

// powDecimal computes base^n for decimal using integer exponentiation.
func powDecimal(base decimal.Decimal, n int) decimal.Decimal {
	result := decimal.NewFromInt(1)
	for i := 0; i < n; i++ {
		result = result.Mul(base)
	}
	return result
}
