package types

import (
	"time"

	"github.com/shopspring/decimal"
)

// ContractAttributes defines all ACTUS contract properties according to v1.1 specification
type ContractAttributes struct {
	// ===== Basic Identification =====
	ContractID   string `json:"contractID"`
	ContractType string `json:"contractType" binding:"required,oneof=PAM LAM ANN SWAPS OPTNS CSH CLM UMP NAM LAX FXOUT STK COM SWPPV CAPFL FUTUR CEG CEC"`
	ContractRole string `json:"contractRole" binding:"required,oneof=RPA RPL LG ST BUY SEL RFL PFL"`

	// ===== Date Attributes =====
	StatusDate            time.Time  `json:"statusDate"`
	InitialExchangeDate   time.Time  `json:"initialExchangeDate" binding:"required"`
	MaturityDate          *time.Time `json:"maturityDate,omitempty"`
	PurchaseDate          *time.Time `json:"purchaseDate,omitempty"`
	TerminationDate       *time.Time `json:"terminationDate,omitempty"`
	CapitalizationEndDate *time.Time `json:"capitalizationEndDate,omitempty"`
	AmortizationDate      *time.Time `json:"amortizationDate,omitempty"` // For ANN contracts

	// ===== Amount Attributes =====
	NotionalPrincipal      decimal.Decimal `json:"notionalPrincipal" binding:"required"`
	NotionalPrincipal2     decimal.Decimal `json:"notionalPrincipal2"` // For FXOUT: notional in currency2
	NominalInterestRate    decimal.Decimal `json:"nominalInterestRate"`
	NominalInterestRate2   decimal.Decimal `json:"nominalInterestRate2"` // For SWPPV: floating leg initial rate
	PremiumDiscountAtIED   decimal.Decimal `json:"premiumDiscountAtIED"`
	PriceAtPurchaseDate    decimal.Decimal `json:"priceAtPurchaseDate"`
	PriceAtTerminationDate decimal.Decimal `json:"priceAtTerminationDate"`

	// ===== Cycle Attributes (Interest Payment) =====
	CycleAnchorDateOfInterestPayment *time.Time `json:"cycleAnchorDateOfInterestPayment,omitempty"`
	CycleOfInterestPayment           string     `json:"cycleOfInterestPayment,omitempty"` // e.g., "P3M"

	// ===== Cycle Attributes (Principal Redemption) =====
	CycleAnchorDateOfPrincipalRedemption *time.Time      `json:"cycleAnchorDateOfPrincipalRedemption,omitempty"`
	CycleOfPrincipalRedemption           string          `json:"cycleOfPrincipalRedemption,omitempty"`
	NextPrincipalRedemptionPayment       decimal.Decimal `json:"nextPrincipalRedemptionPayment"`

	// ===== Cycle Attributes (Rate Reset) =====
	CycleAnchorDateOfRateReset  *time.Time      `json:"cycleAnchorDateOfRateReset,omitempty"`
	CycleOfRateReset            string          `json:"cycleOfRateReset,omitempty"`
	RateSpread                  decimal.Decimal `json:"rateSpread"`
	RateMultiplier              decimal.Decimal `json:"rateMultiplier"`
	MarketObjectCodeOfRateReset string          `json:"marketObjectCodeOfRateReset,omitempty"`
	NextResetRate               decimal.Decimal `json:"nextResetRate"` // Fixed rate for first RR event (RRF)

	// ===== Cycle Attributes (Scaling Index) =====
	CycleAnchorDateOfScalingIndex  *time.Time      `json:"cycleAnchorDateOfScalingIndex,omitempty"`
	CycleOfScalingIndex            string          `json:"cycleOfScalingIndex,omitempty"`
	ScalingEffect                  string          `json:"scalingEffect,omitempty"` // "OOO", "IOO", "ONO", "INO", "OOM", "IOM", "ONM", "INM"
	ScalingIndexAtStatusDate       decimal.Decimal `json:"scalingIndexAtStatusDate"`
	MarketObjectCodeOfScalingIndex string          `json:"marketObjectCodeOfScalingIndex,omitempty"`

	// ===== Date Conventions =====
	DayCountConvention    DayCountConvention    `json:"dayCountConvention" binding:"required"`
	BusinessDayConvention BusinessDayConvention `json:"businessDayConvention"`
	EndOfMonthConvention  EndOfMonthConvention  `json:"endOfMonthConvention"`
	Calendar              string                `json:"calendar"` // e.g., "NoHoliday", "MondayToFriday"

	// ===== Fee Attributes =====
	FeeRate                     decimal.Decimal `json:"feeRate"`
	FeeAccrued                  decimal.Decimal `json:"feeAccrued"`
	CycleAnchorDateOfFee        *time.Time      `json:"cycleAnchorDateOfFee,omitempty"`
	CycleOfFee                  string          `json:"cycleOfFee,omitempty"`
	FeeBasis                    string          `json:"feeBasis,omitempty"`                    // "A"=Absolute, "N"=Notional
	MarketObjectCodeOfDividends string          `json:"marketObjectCodeOfDividends,omitempty"` // For STK contracts

	// ===== Currency =====
	Currency           string `json:"currency" binding:"required,len=3"`
	Currency2          string `json:"currency2,omitempty"` // For FXOUT: second currency
	SettlementCurrency string `json:"settlementCurrency,omitempty"`

	// ===== State Variable Initial Values =====
	AccruedInterest     decimal.Decimal `json:"accruedInterest"`
	ContractPerformance string          `json:"contractPerformance"` // "PF"=Performant, "DL"=Delayed, "DQ"=Delinquent, "DF"=Default

	// ===== Array Attributes (for LAX etc.) =====
	ArrayCycleAnchorDateOfPrincipalRedemption []time.Time       `json:"arrayCycleAnchorDateOfPrincipalRedemption,omitempty"`
	ArrayCycleOfPrincipalRedemption           []string          `json:"arrayCycleOfPrincipalRedemption,omitempty"`
	ArrayNextPrincipalRedemptionPayment       []decimal.Decimal `json:"arrayNextPrincipalRedemptionPayment,omitempty"`
	ArrayIncreaseDecrease                     []string          `json:"arrayIncreaseDecrease,omitempty"` // "INC" or "DEC"
	ArrayCycleAnchorDateOfInterestPayment     []time.Time       `json:"arrayCycleAnchorDateOfInterestPayment,omitempty"`
	ArrayCycleOfInterestPayment               []string          `json:"arrayCycleOfInterestPayment,omitempty"`
	ArrayCycleAnchorDateOfRateReset           []time.Time       `json:"arrayCycleAnchorDateOfRateReset,omitempty"`
	ArrayRate                                 []decimal.Decimal `json:"arrayRate,omitempty"`
	ArrayFixedVariable                        []string          `json:"arrayFixedVariable,omitempty"` // "FIX" or "VAR"

	// ===== Interest Calculation Base (for LAM) =====
	InterestCalculationBase                  string          `json:"interestCalculationBase,omitempty"` // "NT", "NTL", or "NTIED"
	InterestCalculationBaseAmount            decimal.Decimal `json:"interestCalculationBaseAmount"`
	CycleAnchorDateOfInterestCalculationBase *time.Time      `json:"cycleAnchorDateOfInterestCalculationBase,omitempty"`
	CycleOfInterestCalculationBase           string          `json:"cycleOfInterestCalculationBase,omitempty"`

	// ===== Composite Contract Attributes (for SWAPS, OPTNS) =====
	ContractStructure []ContractReference `json:"contractStructure,omitempty"`

	// ===== Options-Specific Attributes =====
	OptionType         OptionType      `json:"optionType,omitempty"`         // "CALL" or "PUT"
	OptionExerciseType string          `json:"optionExerciseType,omitempty"` // "E" (European) or "A" (American)
	OptionStrike1      decimal.Decimal `json:"optionStrike1"`                // Strike price
	SettlementPeriod   string          `json:"settlementPeriod,omitempty"`   // Period between exercise and settlement

	// ===== FXOUT-Specific Attributes =====
	DeliverySettlement string `json:"deliverySettlement,omitempty"` // "D" (Physical delivery) or "S" (Cash settlement)

	// ===== CLM-Specific Attributes =====
	XDayNotice string `json:"xDayNotice,omitempty"` // Notice period for early termination (e.g., "P31D")

	// ===== CEG-Specific Attributes =====
	CoverageOfCreditEnhancement decimal.Decimal `json:"coverageOfCreditEnhancement"`      // Coverage ratio (0.0 to 1.0)
	CreditEventTypeCovered      string          `json:"creditEventTypeCovered,omitempty"` // Type of credit event covered (e.g., "DF", "DQ")
	GuaranteedExposure          string          `json:"guaranteedExposure,omitempty"`     // "NO" (Notional Only) or "NI" (Notional + Interest)

	// ===== Futures-Specific Attributes =====
	FuturesPrice decimal.Decimal `json:"futuresPrice"` // Opening futures price

	// ===== CAPFL-Specific Attributes =====
	LifeCap   decimal.Decimal `json:"lifeCap"`   // Interest rate cap
	LifeFloor decimal.Decimal `json:"lifeFloor"` // Interest rate floor
}

// ContractReference defines a reference to another contract or market object
type ContractReference struct {
	Object      string `json:"object"` // "Contract" or "MarketObject"
	Type        string `json:"type"`
	Role        string `json:"role"` // "FirstLeg", "SecondLeg", "Underlying", etc.
	ReferenceID string `json:"referenceId"`
}

// ===== Enumeration Types =====

// DayCountConvention defines how to calculate year fractions
type DayCountConvention string

const (
	DCC_A_360  DayCountConvention = "A/360"   // Actual/360
	DCC_A_365  DayCountConvention = "A/365"   // Actual/365
	DCC_30E360 DayCountConvention = "30E/360" // 30E/360 (Eurobond)
	DCC_30_360 DayCountConvention = "30/360"  // 30/360 (US)
	DCC_A_A    DayCountConvention = "A/A"     // Actual/Actual (ISDA)
)

// BusinessDayConvention defines how to adjust non-business days
type BusinessDayConvention string

const (
	BDC_NULL BusinessDayConvention = "NULL" // No adjustment
	BDC_SCF  BusinessDayConvention = "SCF"  // Shift/Calculate Following
	BDC_SCMF BusinessDayConvention = "SCMF" // Shift/Calculate Modified Following
	BDC_CSF  BusinessDayConvention = "CSF"  // Calculate/Shift Following
	BDC_CSMF BusinessDayConvention = "CSMF" // Calculate/Shift Modified Following
	BDC_SCP  BusinessDayConvention = "SCP"  // Shift/Calculate Preceding
	BDC_SCMP BusinessDayConvention = "SCMP" // Shift/Calculate Modified Preceding
	BDC_CSP  BusinessDayConvention = "CSP"  // Calculate/Shift Preceding
	BDC_CSMP BusinessDayConvention = "CSMP" // Calculate/Shift Modified Preceding
)

// EndOfMonthConvention defines end-of-month handling
type EndOfMonthConvention string

const (
	EOMC_EOM EndOfMonthConvention = "EOM" // End Of Month
	EOMC_SD  EndOfMonthConvention = "SD"  // Same Day
)

// OptionType defines the type of option contract
type OptionType string

const (
	OT_CALL OptionType = "CALL" // Call option - right to buy
	OT_PUT  OptionType = "PUT"  // Put option - right to sell
)
