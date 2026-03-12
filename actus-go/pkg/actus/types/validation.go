package types

import (
	"fmt"
	"time"

	"github.com/shopspring/decimal"
)

// ===== Security Limit Constants =====
// These limits are designed to prevent numerical overflow and catch data entry errors
const (
	// Interest rate limits (as decimal, e.g., 0.10 = 10%)
	// Range: -50% to +200% covers most legitimate financial scenarios
	// including negative rates (some central banks) and high-yield instruments
	MinInterestRate = -0.50 // -50%
	MaxInterestRate = 2.00  // +200%

	// Principal amount limits
	// Maximum: 999 trillion (covers largest sovereign debt instruments)
	MaxPrincipalAmount = 999_999_999_999_999.0

	// Date validation limits
	// Maximum years from current time for contract dates
	MaxYearsPast   = 100
	MaxYearsFuture = 100
)

// Security limit decimal values (initialized once)
var (
	minInterestRateDecimal = decimal.NewFromFloat(MinInterestRate)
	maxInterestRateDecimal = decimal.NewFromFloat(MaxInterestRate)
	maxPrincipalDecimal    = decimal.NewFromFloat(MaxPrincipalAmount)
)

// Validate checks the contract attributes for completeness and consistency
func (ca *ContractAttributes) Validate() error {
	// Basic field validation
	if ca.ContractID == "" {
		return fmt.Errorf("contractID is required")
	}

	if ca.ContractType == "" {
		return fmt.Errorf("contractType is required")
	}

	if ca.ContractRole == "" {
		return fmt.Errorf("contractRole is required")
	}

	// Date logic validation
	if ca.InitialExchangeDate.IsZero() {
		return fmt.Errorf("initialExchangeDate is required")
	}

	if ca.MaturityDate != nil && ca.MaturityDate.Before(ca.InitialExchangeDate) {
		return fmt.Errorf("maturityDate (%s) must be after initialExchangeDate (%s)",
			ca.MaturityDate.Format("2006-01-02"), ca.InitialExchangeDate.Format("2006-01-02"))
	}

	// Amount validation
	if ca.NotionalPrincipal.IsZero() {
		return fmt.Errorf("notionalPrincipal cannot be zero")
	}

	// ===== SECURITY CHECKS =====

	// 1. Principal amount upper limit check
	if ca.NotionalPrincipal.Abs().GreaterThan(maxPrincipalDecimal) {
		return fmt.Errorf("notionalPrincipal exceeds maximum allowed value: %s (max: %s)",
			ca.NotionalPrincipal.String(), maxPrincipalDecimal.String())
	}

	// 2. Interest rate range validation (only if non-zero)
	if !ca.NominalInterestRate.IsZero() {
		if ca.NominalInterestRate.LessThan(minInterestRateDecimal) {
			return fmt.Errorf("nominalInterestRate is below minimum allowed: %s (min: %.0f%%)",
				ca.NominalInterestRate.String(), MinInterestRate*100)
		}
		if ca.NominalInterestRate.GreaterThan(maxInterestRateDecimal) {
			return fmt.Errorf("nominalInterestRate exceeds maximum allowed: %s (max: %.0f%%)",
				ca.NominalInterestRate.String(), MaxInterestRate*100)
		}
	}

	// 3. Date range validation (truncate to date precision to avoid time-of-day skew)
	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	minDate := today.AddDate(-MaxYearsPast, 0, 0)
	maxDate := today.AddDate(MaxYearsFuture, 0, 0)

	if err := validateDateInRange(ca.InitialExchangeDate, minDate, maxDate, "initialExchangeDate"); err != nil {
		return err
	}
	if ca.MaturityDate != nil {
		if err := validateDateInRange(*ca.MaturityDate, minDate, maxDate, "maturityDate"); err != nil {
			return err
		}
	}
	if ca.StatusDate.IsZero() == false {
		if err := validateDateInRange(ca.StatusDate, minDate, maxDate, "statusDate"); err != nil {
			return err
		}
	}

	// Currency validation
	if len(ca.Currency) != 3 {
		return fmt.Errorf("currency must be 3-letter code (e.g., USD), got: %s", ca.Currency)
	}

	// Contract type specific validation
	switch ca.ContractType {
	case "PAM":
		return ca.validatePAM()
	case "LAM":
		return ca.validateLAM()
	case "ANN":
		return ca.validateANN()
	case "SWAPS":
		return ca.validateSWAPS()
	case "OPTNS":
		return ca.validateOPTNS()
	case "CSH":
		return ca.validateCSH()
	case "CLM":
		return ca.validateCLM()
	case "UMP":
		return ca.validateUMP()
	case "NAM":
		return ca.validateNAM()
	case "LAX":
		return ca.validateLAX()
	case "FXOUT":
		return ca.validateFXOUT()
	case "STK":
		return ca.validateSTK()
	case "COM":
		return ca.validateCOM()
	case "SWPPV":
		return ca.validateSWPPV()
	case "CAPFL":
		return ca.validateCAPFL()
	case "FUTUR":
		return ca.validateFUTUR()
	case "CEG":
		return ca.validateCEG()
	case "CEC":
		return ca.validateCEC()
	default:
		return fmt.Errorf("unsupported contract type: %s", ca.ContractType)
	}
}

// validatePAM validates PAM-specific requirements
func (ca *ContractAttributes) validatePAM() error {
	// PAM requires maturity date
	if ca.MaturityDate == nil {
		return fmt.Errorf("PAM contract requires maturityDate")
	}

	// PAM should not have PR cycle (principal redeemed at maturity)
	if ca.CycleOfPrincipalRedemption != "" {
		return fmt.Errorf("PAM contract should not have principalRedemption cycle")
	}

	return nil
}

// validateLAM validates LAM-specific requirements
func (ca *ContractAttributes) validateLAM() error {
	// LAM requires PR cycle
	if ca.CycleOfPrincipalRedemption == "" {
		return fmt.Errorf("LAM contract requires principalRedemption cycle")
	}

	// Note: MaturityDate is optional - can be set during testing via "to" field
	// or determined by termination date. Some test cases don't specify maturityDate.

	// If PRNXT is not specified, it will be calculated automatically
	// So no validation needed for NextPrincipalRedemptionPayment

	return nil
}

// validateANN validates ANN-specific requirements
func (ca *ContractAttributes) validateANN() error {
	// ANN requires maturity date
	if ca.MaturityDate == nil {
		return fmt.Errorf("ANN contract requires maturityDate")
	}

	// ANN requires PR cycle
	if ca.CycleOfPrincipalRedemption == "" {
		return fmt.Errorf("ANN contract requires principalRedemption cycle")
	}

	// ANN requires interest payment cycle
	if ca.CycleOfInterestPayment == "" {
		return fmt.Errorf("ANN contract requires interestPayment cycle")
	}

	return nil
}

// validateSWAPS validates SWAPS-specific requirements
func (ca *ContractAttributes) validateSWAPS() error {
	// SWAPS implementation uses simplified single-contract architecture
	// Validation is handled in swaps.NewSWAPS()

	// Basic checks that should always pass if NewSWAPS() succeeded
	if ca.ContractRole != "RFL" && ca.ContractRole != "PFL" {
		return fmt.Errorf("SWAPS ContractRole must be RFL or PFL, got %s", ca.ContractRole)
	}

	if ca.CycleOfInterestPayment == "" {
		return fmt.Errorf("SWAPS requires CycleOfInterestPayment")
	}

	return nil
}

// validateOPTNS validates OPTNS-specific requirements
func (ca *ContractAttributes) validateOPTNS() error {
	// OPTNS implementation uses simplified single-contract architecture
	// Validation is handled in optns.NewOPTNS()

	// Basic checks that should always pass if NewOPTNS() succeeded
	if ca.ContractRole != "BUY" && ca.ContractRole != "SEL" {
		return fmt.Errorf("OPTNS ContractRole must be BUY or SEL, got %s", ca.ContractRole)
	}

	if ca.MaturityDate == nil {
		return fmt.Errorf("OPTNS requires MaturityDate (exercise date)")
	}

	return nil
}

// validateCSH validates CSH-specific requirements
func (ca *ContractAttributes) validateCSH() error {
	// CSH is the simplest contract type - represents cash holdings
	// No maturity date required
	// No interest payment or principal redemption cycles
	// No special validation needed beyond basic checks

	return nil
}

// validateCLM validates CLM-specific requirements
func (ca *ContractAttributes) validateCLM() error {
	// CLM is a Call Money contract (demand deposit/loan)
	// Can have optional maturity date
	// Daily interest accrual using nominal interest rate
	// No strict requirements beyond basic validation

	return nil
}

// validateUMP validates UMP-specific requirements
func (ca *ContractAttributes) validateUMP() error {
	// UMP is an Undefined Maturity Profile contract (credit card, revolving credit)
	// Event-driven with no fixed repayment schedule
	// Dynamic maturity based on principal balance
	// No strict requirements beyond basic validation

	return nil
}

// validateNAM validates NAM-specific requirements
func (ca *ContractAttributes) validateNAM() error {
	// NAM is a Negative Amortizer contract (graduated payment mortgage, student loans)
	// Payment less than accrued interest causes principal to grow
	// Requires maturity date and principal redemption payment amount
	// Maturity date required
	if ca.MaturityDate == nil {
		return fmt.Errorf("NAM contract requires maturityDate")
	}

	// Principal redemption cycle required
	if ca.CycleOfPrincipalRedemption == "" {
		return fmt.Errorf("NAM contract requires principalRedemption cycle")
	}

	return nil
}

// validateLAX validates LAX-specific requirements
func (ca *ContractAttributes) validateLAX() error {
	// LAX is an Exotic Linear Amortizer with array-based schedules
	// Supports variable principal redemption patterns (DEC/INC)
	// Requires maturity date and array attributes

	// Maturity date required
	if ca.MaturityDate == nil {
		return fmt.Errorf("LAX contract requires maturityDate")
	}

	// Principal redemption cycle required
	if ca.CycleOfPrincipalRedemption == "" {
		return fmt.Errorf("LAX contract requires principalRedemption cycle")
	}

	// Array attributes required for exotic schedules
	if ca.ArrayCycleAnchorDateOfPrincipalRedemption == nil ||
		len(ca.ArrayCycleAnchorDateOfPrincipalRedemption) == 0 {
		return fmt.Errorf("LAX contract requires arrayCycleAnchorDateOfPrincipalRedemption")
	}

	if ca.ArrayNextPrincipalRedemptionPayment == nil ||
		len(ca.ArrayNextPrincipalRedemptionPayment) == 0 {
		return fmt.Errorf("LAX contract requires arrayNextPrincipalRedemptionPayment")
	}

	// Array lengths should match
	if len(ca.ArrayCycleAnchorDateOfPrincipalRedemption) != len(ca.ArrayNextPrincipalRedemptionPayment) {
		return fmt.Errorf("LAX array lengths must match: anchors=%d, payments=%d",
			len(ca.ArrayCycleAnchorDateOfPrincipalRedemption),
			len(ca.ArrayNextPrincipalRedemptionPayment))
	}

	// If ArrayIncreaseDecrease is provided, it should match array length
	if ca.ArrayIncreaseDecrease != nil &&
		len(ca.ArrayIncreaseDecrease) != len(ca.ArrayCycleAnchorDateOfPrincipalRedemption) {
		return fmt.Errorf("LAX arrayIncreaseDecrease length must match anchor dates length")
	}

	return nil
}

// validateFXOUT validates FXOUT-specific requirements
func (ca *ContractAttributes) validateFXOUT() error {
	// FXOUT is a Foreign Exchange Outright (forward) contract
	// Requires maturity date (settlement date)
	if ca.MaturityDate == nil {
		return fmt.Errorf("FXOUT contract requires maturityDate (settlement date)")
	}

	// Requires settlement currency
	if ca.SettlementCurrency == "" {
		return fmt.Errorf("FXOUT contract requires settlementCurrency")
	}

	return nil
}

// validateSTK validates STK-specific requirements
func (ca *ContractAttributes) validateSTK() error {
	// STK is a Stock contract
	// No strict requirements - can have dividends or not
	// No maturity date required (perpetual)
	return nil
}

// validateCOM validates COM-specific requirements
func (ca *ContractAttributes) validateCOM() error {
	// COM is a Commodity contract
	// Similar to STK - can have maturity or not
	// No strict requirements
	return nil
}

// validateSWPPV validates SWPPV-specific requirements
func (ca *ContractAttributes) validateSWPPV() error {
	// SWPPV is a Plain Vanilla Interest Rate Swap
	// Requires maturity date
	if ca.MaturityDate == nil {
		return fmt.Errorf("SWPPV contract requires maturityDate")
	}

	// Requires interest payment cycle
	if ca.CycleOfInterestPayment == "" {
		return fmt.Errorf("SWPPV contract requires CycleOfInterestPayment")
	}

	// Requires fixed rate (stored in NominalInterestRate)
	if ca.NominalInterestRate.IsZero() {
		return fmt.Errorf("SWPPV contract requires NominalInterestRate (fixed leg rate)")
	}

	// Contract role should be RFL (Receive Fixed Leg) or PFL (Pay Fixed Leg)
	// Also accept short forms RF and PF for ACTUS test compatibility
	role := ca.ContractRole
	if role != "RFL" && role != "PFL" && role != "RF" && role != "PF" {
		return fmt.Errorf("SWPPV ContractRole must be RFL/RF (Receive Fixed) or PFL/PF (Pay Fixed), got %s", ca.ContractRole)
	}

	// Normalize to long form
	if role == "RF" {
		ca.ContractRole = "RFL"
	} else if role == "PF" {
		ca.ContractRole = "PFL"
	}

	return nil
}

// validateCAPFL validates CAPFL-specific requirements
func (ca *ContractAttributes) validateCAPFL() error {
	// CAPFL is a Cap-Floor interest rate derivative
	// Requires maturity date
	if ca.MaturityDate == nil {
		return fmt.Errorf("CAPFL contract requires maturityDate")
	}

	// Requires interest payment cycle (rate check schedule)
	if ca.CycleOfInterestPayment == "" {
		return fmt.Errorf("CAPFL contract requires CycleOfInterestPayment")
	}

	// At least one of cap rate or floor rate must be specified
	// Cap rate stored in NominalInterestRate
	// Floor rate stored in RateSpread
	if ca.NominalInterestRate.IsZero() && ca.RateSpread.IsZero() {
		return fmt.Errorf("CAPFL contract requires at least cap rate (NominalInterestRate) or floor rate (RateSpread)")
	}

	// Contract role should be BUY or SEL
	if ca.ContractRole != "BUY" && ca.ContractRole != "SEL" {
		return fmt.Errorf("CAPFL ContractRole must be BUY or SEL, got %s", ca.ContractRole)
	}

	return nil
}

// validateFUTUR validates FUTUR-specific requirements
func (ca *ContractAttributes) validateFUTUR() error {
	// FUTUR is a futures contract
	// Requires maturity date (expiry date)
	if ca.MaturityDate == nil {
		return fmt.Errorf("FUTUR contract requires maturityDate (expiry date)")
	}

	// Requires futures opening price (stored in NominalInterestRate)
	if ca.NominalInterestRate.IsZero() {
		return fmt.Errorf("FUTUR contract requires NominalInterestRate (futures opening price)")
	}

	// Contract role should be BUY, SEL, LG, or ST
	if ca.ContractRole != "BUY" && ca.ContractRole != "SEL" &&
		ca.ContractRole != "LG" && ca.ContractRole != "ST" {
		return fmt.Errorf("FUTUR ContractRole must be BUY, SEL, LG, or ST, got %s", ca.ContractRole)
	}

	return nil
}

// validateCEG validates CEG-specific requirements
func (ca *ContractAttributes) validateCEG() error {
	// CEG is a Credit Enhancement Guarantee
	// Requires maturity date (guarantee expiry)
	if ca.MaturityDate == nil {
		return fmt.Errorf("CEG contract requires maturityDate (guarantee expiry)")
	}

	// Requires fee rate for guarantee premium
	if ca.FeeRate.IsZero() {
		return fmt.Errorf("CEG contract requires feeRate (guarantee premium rate)")
	}

	// Contract role should be BUY or SEL
	if ca.ContractRole != "BUY" && ca.ContractRole != "SEL" {
		return fmt.Errorf("CEG ContractRole must be BUY or SEL, got %s", ca.ContractRole)
	}

	return nil
}

// validateCEC validates CEC-specific requirements
func (ca *ContractAttributes) validateCEC() error {
	// CEC is a Credit Enhancement Collateral
	// Requires maturity date (collateral return date)
	if ca.MaturityDate == nil {
		return fmt.Errorf("CEC contract requires maturityDate (collateral return date)")
	}

	// Contract role should be BUY or SEL
	if ca.ContractRole != "BUY" && ca.ContractRole != "SEL" {
		return fmt.Errorf("CEC ContractRole must be BUY or SEL, got %s", ca.ContractRole)
	}

	return nil
}

// SetDefaults sets default values for optional attributes
func (ca *ContractAttributes) SetDefaults() {
	// Set default status date if not specified
	if ca.StatusDate.IsZero() {
		ca.StatusDate = ca.InitialExchangeDate
	}

	// Set default business day convention
	if ca.BusinessDayConvention == "" {
		ca.BusinessDayConvention = BDC_NULL
	}

	// Set default end-of-month convention
	if ca.EndOfMonthConvention == "" {
		ca.EndOfMonthConvention = EOMC_SD
	}

	// Set default calendar
	if ca.Calendar == "" {
		ca.Calendar = "NoHoliday"
	}

	// Set default contract performance
	if ca.ContractPerformance == "" {
		ca.ContractPerformance = "PF" // Performant
	}

	// Set default rate multiplier if rate reset is enabled
	if ca.MarketObjectCodeOfRateReset != "" && ca.RateMultiplier.IsZero() {
		ca.RateMultiplier = decimal.NewFromInt(1)
	}

	// Set default fee basis
	if ca.FeeBasis == "" && !ca.FeeRate.IsZero() {
		ca.FeeBasis = "N" // Notional-based
	}
}

// IsZeroTime checks if a time pointer is nil or zero
func IsZeroTime(t *time.Time) bool {
	return t == nil || t.IsZero()
}

// validateDateInRange checks if a date is within the specified min/max range
// Returns an error if the date is outside the valid range
func validateDateInRange(date time.Time, minDate, maxDate time.Time, fieldName string) error {
	// Truncate to date precision — contract dates are day-granularity
	dateDay := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, date.Location())
	if dateDay.Before(minDate) {
		return fmt.Errorf("%s is too far in the past (more than %d years): %s",
			fieldName, MaxYearsPast, date.Format("2006-01-02"))
	}
	if dateDay.After(maxDate) {
		return fmt.Errorf("%s is too far in the future (more than %d years): %s",
			fieldName, MaxYearsFuture, date.Format("2006-01-02"))
	}
	return nil
}
