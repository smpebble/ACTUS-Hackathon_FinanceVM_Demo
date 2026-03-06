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

// BondScenario demonstrates a 5-year corporate bond with semi-annual coupon payments.
type BondScenario struct{}

func (s *BondScenario) Name() string             { return "Corporate Bond (5Y, 3.5%, Semi-Annual, Act/360)" }
func (s *BondScenario) Type() model.ScenarioType { return model.ScenarioBond }

func (s *BondScenario) Run(ctx *ScenarioContext, params map[string]interface{}) (*model.ScenarioResult, error) {
	result := &model.ScenarioResult{
		Type:        model.ScenarioBond,
		Name:        s.Name(),
		Description: "5-year corporate bond, 100M TWD, 3.5% coupon, semi-annual payments, Act/360 day count.",
		StartTime:   time.Now(),
	}

	// Bond parameters
	issueDate := time.Date(2025, 1, 15, 0, 0, 0, 0, time.UTC)
	maturityDate := time.Date(2030, 1, 15, 0, 0, 0, 0, time.UTC)

	notionalVal := getParamFloat(params, "notional", 100000000.0) // 100M TWD
	notional := decimal.NewFromFloat(notionalVal)
	rateVal := getParamFloat(params, "rate", 0.035) // 3.5%
	couponRate := decimal.NewFromFloat(rateVal)

	attrs := &types.ContractAttributes{
		ContractID:                       "BOND-TW-2025-001",
		ContractType:                     "PAM",
		ContractRole:                     "RPA",
		Currency:                         "TWD",
		DayCountConvention:               dayCountA360(),
		StatusDate:                       issueDate.AddDate(0, 0, -1),
		InitialExchangeDate:              issueDate,
		MaturityDate:                     makeTimePtr(maturityDate),
		NotionalPrincipal:                notional,
		NominalInterestRate:              couponRate,
		CycleOfInterestPayment:           "P6M",
		CycleAnchorDateOfInterestPayment: makeTimePtr(time.Date(2025, 7, 15, 0, 0, 0, 0, time.UTC)),
	}

	contract, schedule, state, err := ctx.ACTUS.CreatePAMContract(attrs)
	if err != nil {
		return nil, fmt.Errorf("bond PAM contract creation failed: %w", err)
	}

	// Simulate cashflows
	rf := adapter.NewStaticRiskFactorObserver(nil)
	cashflows, err := ctx.ACTUS.SimulateCashflows(contract, contract, schedule, state, rf)
	if err != nil {
		return nil, fmt.Errorf("bond cashflow simulation failed: %w", err)
	}
	result.CashFlowEvents = cashflows

	// Precision test: Act/360 interest calculation
	// First coupon period: 2025-01-15 to 2025-07-15 = 181 days
	firstCouponStart := issueDate
	firstCouponEnd := time.Date(2025, 7, 15, 0, 0, 0, 0, time.UTC)
	days := firstCouponEnd.Sub(firstCouponStart).Hours() / 24

	// Decimal calculation
	decInterest := notional.Mul(couponRate).Mul(decimal.NewFromFloat(days).Div(decimal.NewFromInt(360)))
	// Float calculation
	floatInterest := notionalVal * rateVal * (days / 360.0)

	result.PrecisionTests = append(result.PrecisionTests,
		comparePrecision(
			fmt.Sprintf("First coupon (%.0f days, Act/360): 100M x 3.5%% x %.0f/360", days, days),
			decInterest,
			floatInterest,
		),
	)

	// Second coupon period precision test
	secondCouponEnd := time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC)
	days2 := secondCouponEnd.Sub(firstCouponEnd).Hours() / 24
	decInterest2 := notional.Mul(couponRate).Mul(decimal.NewFromFloat(days2).Div(decimal.NewFromInt(360)))
	floatInterest2 := notionalVal * rateVal * (days2 / 360.0)

	result.PrecisionTests = append(result.PrecisionTests,
		comparePrecision(
			fmt.Sprintf("Second coupon (%.0f days, Act/360): 100M x 3.5%% x %.0f/360", days2, days2),
			decInterest2,
			floatInterest2,
		),
	)

	// FVM instrument lifecycle
	inst := ctx.FVM.CreateInstrument(
		"TW Corporate Bond 2025-A",
		"5-year corporate bond, 3.5% semi-annual coupon",
		"AIMICHIA TECHNOLOGY CO., LTD.",
		model.CurrencyTWD,
		"PAM",
		model.TokenStandardERC1400,
	)
	inst.ISIN = "TW0000000001"
	inst.InterestRate = couponRate
	if err := ctx.FVM.IssueInstrument(inst.ID, notional, issueDate); err != nil {
		return nil, err
	}
	if err := ctx.FVM.ActivateInstrument(inst.ID); err != nil {
		return nil, err
	}
	inst.MaturityDate = makeTimePtr(maturityDate)
	result.Instrument = inst

	// Generate Solidity ERC-1400 Security Token contract
	outputDir := filepath.Join(ctx.BaseDir, "generated", "bond")
	solidityFiles, err := ctx.Codegen.GenerateSolidityContract("CorporateBondToken", "PAM", schedule, outputDir)
	if err != nil {
		return nil, fmt.Errorf("bond solidity generation failed: %w", err)
	}
	result.SolidityFiles = solidityFiles

	// Settlement, accounting, ISO 20022, vLEI
	verifyAndSettle(ctx, result)

	// Attach Risk Analysis
	result.RiskAnalysis = CalculateRiskAnalysis(result.CashFlowEvents, "PAM", params)

	result.EndTime = time.Now()
	return result, nil
}
