package ann

import (
	"fmt"
	"time"

	"github.com/shopspring/decimal"
	"github.com/yourusername/actus-go/pkg/actus/contracts/pam"
	"github.com/yourusername/actus-go/pkg/actus/events"
	"github.com/yourusername/actus-go/pkg/actus/states"
	"github.com/yourusername/actus-go/pkg/actus/types"
	"github.com/yourusername/actus-go/pkg/actus/utils"
	"github.com/yourusername/actus-go/pkg/riskfactor"
)

// ANN represents an Annuity contract
// This is a loan with constant periodic payments (principal + interest)
// where the interest portion decreases and principal portion increases over time
type ANN struct {
	*pam.PAM // Embed PAM to inherit common functionality
}

// NewANN creates a new ANN contract instance
func NewANN(attrs *types.ContractAttributes) (*ANN, error) {
	if attrs == nil {
		return nil, fmt.Errorf("contract attributes cannot be nil")
	}

	// Set defaults
	attrs.SetDefaults()

	// Validate attributes
	if err := attrs.Validate(); err != nil {
		return nil, fmt.Errorf("ANN validation failed: %w", err)
	}

	// Create embedded PAM
	pamContract, err := pam.NewPAM(attrs)
	if err != nil {
		return nil, err
	}

	return &ANN{
		PAM: pamContract,
	}, nil
}

// InitializeState creates the initial contract state for ANN
func (a *ANN) InitializeState() (*states.ContractState, error) {
	state, err := states.InitializeState(a.Attributes)
	if err != nil {
		return nil, err
	}

	// ANN-specific: Calculate default annuity payment if not specified
	roleSign := a.getRoleSign()
	if a.Attributes.NextPrincipalRedemptionPayment.IsZero() {
		// Calculate annuity payment (always positive value)
		// Then apply role sign to get correct sign
		basePayment := a.calculateAnnuityPayment(state)
		state.NextPrincipalPayment = roleSign.Mul(basePayment)
	} else {
		state.NextPrincipalPayment = a.Attributes.NextPrincipalRedemptionPayment
	}

	// Set interest calculation base
	if a.Attributes.InterestCalculationBase == "NTIED" && !a.Attributes.InterestCalculationBaseAmount.IsZero() {
		state.InterestCalculationBase = roleSign.Mul(a.Attributes.InterestCalculationBaseAmount)
	} else {
		// Default: use current notional principal
		state.InterestCalculationBase = state.NotionalPrincipal
	}

	return state, nil
}

// calculateAnnuityPayment calculates the constant payment amount for an annuity
// using the ACTUS standard formula (Section 3.8):
// A(s,T,n,a,r) = (n+a) × ∏[1 + r×Y(ti,ti+1)] / [1 + ∑∏[1 + r×Y(tj,tj+1)]]
// Where n = principal, a = accrued interest, r = annual rate, Y = year fraction
func (a *ANN) calculateAnnuityPayment(state *states.ContractState) decimal.Decimal {
	cycle, err := utils.ParseCycle(a.Attributes.CycleOfPrincipalRedemption)
	if err != nil || cycle == nil {
		return decimal.Zero
	}

	// Determine anchor date and end date
	anchor := a.Attributes.InitialExchangeDate
	usePRanchor := false
	if a.Attributes.CycleAnchorDateOfPrincipalRedemption != nil {
		anchor = *a.Attributes.CycleAnchorDateOfPrincipalRedemption
		usePRanchor = true
	}

	// CRITICAL: For annuity calculation, use AmortizationDate (not MaturityDate)
	// The ACTUS formula calculates the annuity payment to fully amortize by AmortizationDate
	var amd time.Time
	if a.Attributes.AmortizationDate != nil {
		amd = *a.Attributes.AmortizationDate
	} else if a.Attributes.MaturityDate != nil {
		// Fallback to MaturityDate if AmortizationDate not set
		amd = *a.Attributes.MaturityDate
	} else {
		return decimal.Zero
	}

	// Generate PR schedule dates to AmortizationDate
	dates := utils.GenerateCyclicDates(anchor, cycle, amd, false)

	// Determine actual PR event dates (ti values in ACTUS formula)
	var prDates []time.Time
	if usePRanchor {
		// PR anchor: events start at anchor
		prDates = dates
	} else {
		// IED anchor: events start after IED
		if len(dates) > 1 {
			prDates = dates[1:]
		} else {
			return decimal.Zero
		}
	}

	// CRITICAL: For annuity calculation, ensure we include the final period to AMD
	// Even if there's no PR event at AMD, the formula needs to discount cash flows to AMD
	// This is necessary for partially amortizing annuities with balloon payments
	if len(prDates) > 0 && !prDates[len(prDates)-1].Equal(amd) {
		prDates = append(prDates, amd)
	}

	m := len(prDates)
	if m <= 0 {
		return decimal.Zero
	}

	// Get parameters for ACTUS formula
	// A(s,T,n,a,r) where:
	// n = notional principal
	// a = accrued interest as per time s (at PRF event, which is before first PR)
	// r = nominal interest rate
	principal := state.NotionalPrincipal.Abs()
	rate := a.Attributes.NominalInterestRate
	dcc := a.Attributes.DayCountConvention

	// Calculate accrued interest and principal at PRF (before first PR event)
	// This needs to account for:
	// 1. Interest capitalization (IPCI) if capitalizationEndDate is set
	// 2. Interest payments (IP) between IED and PRF
	// 3. Accrued interest from last IP/IPCI event to PRF

	var adjustedPrincipal decimal.Decimal = principal
	var accruedInt decimal.Decimal

	if len(prDates) > 0 {
		firstPR := prDates[0]

		// Step 1: Handle capitalization if capitalizationEndDate is set
		if a.Attributes.CapitalizationEndDate != nil {
			capEnd := *a.Attributes.CapitalizationEndDate

			// Only capitalize if capEnd is before first PR
			if capEnd.Before(firstPR) {
				// Calculate interest from IED to capitalization date
				iedToCapYf := utils.YearFraction(a.Attributes.InitialExchangeDate, capEnd, dcc)
				capitalizedInterest := principal.Mul(rate).Mul(iedToCapYf)

				// Add capitalized interest to principal
				adjustedPrincipal = principal.Add(capitalizedInterest)

				// Step 2: Calculate accrued interest from capEnd to PRF
				// Need to consider if there are IP events between capEnd and firstPR
				lastAccrualDate := capEnd

				// Check if there are IP events before first PR
				if a.Attributes.CycleAnchorDateOfInterestPayment != nil && a.Attributes.CycleOfInterestPayment != "" {
					ipAnchor := *a.Attributes.CycleAnchorDateOfInterestPayment
					ipCycle, err := utils.ParseCycle(a.Attributes.CycleOfInterestPayment)

					if err == nil && ipCycle != nil {
						// Generate IP dates from capEnd to firstPR
						ipDates := utils.GenerateCyclicDates(ipAnchor, ipCycle, firstPR, false)

						// Find the last IP event before firstPR that's after capEnd
						for _, ipDate := range ipDates {
							if ipDate.After(capEnd) && ipDate.Before(firstPR) {
								lastAccrualDate = ipDate
							}
						}
					}
				}

				// Calculate accrued interest from last accrual date to firstPR
				// Note: ACTUS formula uses accrued interest at PRF, which for simplicity
				// we approximate as the interest accrued to firstPR
				lastToPRyf := utils.YearFraction(lastAccrualDate, firstPR, dcc)
				accruedInt = adjustedPrincipal.Mul(rate).Mul(lastToPRyf)
			}
		} else {
			// No capitalization, calculate accrued interest from IED (or last IP) to first PR
			lastAccrualDate := a.Attributes.InitialExchangeDate

			// Check if there are IP events before first PR
			if a.Attributes.CycleAnchorDateOfInterestPayment != nil && a.Attributes.CycleOfInterestPayment != "" {
				ipAnchor := *a.Attributes.CycleAnchorDateOfInterestPayment
				ipCycle, err := utils.ParseCycle(a.Attributes.CycleOfInterestPayment)

				if err == nil && ipCycle != nil {
					// Generate IP dates from IED to firstPR
					ipDates := utils.GenerateCyclicDates(ipAnchor, ipCycle, firstPR, false)

					// Find the last IP event before firstPR
					for _, ipDate := range ipDates {
						if ipDate.After(a.Attributes.InitialExchangeDate) && ipDate.Before(firstPR) {
							lastAccrualDate = ipDate
						}
					}
				}
			}

			// Calculate accrued interest from last accrual date to firstPR
			// Note: ACTUS formula uses accrued interest at PRF, which for simplicity
			// we approximate as the interest accrued to firstPR
			lastToPRyf := utils.YearFraction(lastAccrualDate, firstPR, dcc)
			accruedInt = principal.Mul(rate).Mul(lastToPRyf)
		}

		// Update principal for formula calculation
		principal = adjustedPrincipal
	} else {
		accruedInt = state.AccruedInterest
	}

	// If rate is zero, simple equal payment
	if rate.IsZero() {
		// Safety check: ensure m > 0 to prevent division by zero
		if m <= 0 {
			return decimal.Zero
		}
		return principal.Add(accruedInt).Div(decimal.NewFromInt(int64(m)))
	}

	// Calculate year fractions for each period
	// Y(ti, ti+1) for i = 1 to m-1
	yearFractions := make([]decimal.Decimal, m-1)
	for i := 0; i < m-1; i++ {
		yf := utils.YearFraction(prDates[i], prDates[i+1], dcc)
		yearFractions[i] = yf
	}

	// PRECISION OPTIMIZATION: Increase precision for complex annuity formula
	// This reduces cumulative rounding errors in the nested multiplication/division
	oldPrecision := decimal.DivisionPrecision
	decimal.DivisionPrecision = 32 // Increase from default 16 to 32
	defer func() { decimal.DivisionPrecision = oldPrecision }()

	// Pre-calculate terms: term[i] = 1 + r×Y(ti, ti+1)
	// This avoids repeated calculation in the loops below
	terms := make([]decimal.Decimal, m-1)
	for i := 0; i < m-1; i++ {
		terms[i] = utils.DecimalOne.Add(rate.Mul(yearFractions[i]))
	}

	// Calculate numerator: (n + a) × ∏(i=1 to m-1)[term[i]]
	numeratorProduct := utils.DecimalOne
	for i := 0; i < m-1; i++ {
		numeratorProduct = numeratorProduct.Mul(terms[i])
	}
	numerator := principal.Add(accruedInt).Mul(numeratorProduct)

	// PERFORMANCE OPTIMIZATION: Calculate denominator in O(n) instead of O(n²)
	// Original formula: 1 + ∑(i=1 to m-1)∏(j=i to m-1)[term[j]]
	// Optimization: Use cumulative products from right to left
	// cumulativeProducts[i] = ∏(j=i to m-2)[term[j]]
	// This reduces complexity from O(n²) to O(n)
	cumulativeProducts := make([]decimal.Decimal, m)
	cumulativeProducts[m-1] = utils.DecimalOne
	for j := m - 2; j >= 0; j-- {
		cumulativeProducts[j] = cumulativeProducts[j+1].Mul(terms[j])
	}

	// Sum up cumulative products
	denominatorSum := decimal.Zero
	for i := 0; i < m-1; i++ {
		denominatorSum = denominatorSum.Add(cumulativeProducts[i])
	}
	denominator := utils.DecimalOne.Add(denominatorSum)

	// Calculate payment
	if denominator.IsZero() {
		// Safety check: ensure m > 0 to prevent division by zero
		if m <= 0 {
			return decimal.Zero
		}
		return principal.Add(accruedInt).Div(decimal.NewFromInt(int64(m)))
	}

	payment := numerator.Div(denominator)
	return payment
}

// getRoleSign returns the sign multiplier based on contract role
func (a *ANN) getRoleSign() decimal.Decimal {
	switch a.Attributes.ContractRole {
	case "RPA", "LG", "BUY", "RFL":
		return decimal.NewFromInt(1)
	case "RPL", "ST", "SEL", "PFL":
		return decimal.NewFromInt(-1)
	default:
		return decimal.NewFromInt(1)
	}
}

// recalculateAnnuityPayment calculates the annuity payment for remaining periods
// This is used after Rate Reset events to recalculate with the new interest rate
// Uses: current notional principal from state, current interest rate from state
func (a *ANN) recalculateAnnuityPayment(state *states.ContractState, eventTime time.Time) decimal.Decimal {
	cycle, err := utils.ParseCycle(a.Attributes.CycleOfPrincipalRedemption)
	if err != nil || cycle == nil {
		return state.NextPrincipalPayment
	}

	// Use AmortizationDate or MaturityDate
	var amd time.Time
	if a.Attributes.AmortizationDate != nil {
		amd = *a.Attributes.AmortizationDate
	} else if a.Attributes.MaturityDate != nil {
		amd = *a.Attributes.MaturityDate
	} else {
		return state.NextPrincipalPayment
	}

	// Determine anchor date - use original PR anchor to maintain proper schedule alignment
	anchor := a.Attributes.InitialExchangeDate
	if a.Attributes.CycleAnchorDateOfPrincipalRedemption != nil {
		anchor = *a.Attributes.CycleAnchorDateOfPrincipalRedemption
	}

	// Generate all PR schedule dates from anchor to AmortizationDate
	dates := utils.GenerateCyclicDates(anchor, cycle, amd, false)

	// Filter dates to only include those strictly after eventTime
	var prDates []time.Time
	for _, d := range dates {
		if d.After(eventTime) {
			prDates = append(prDates, d)
		}
	}

	// Include AMD if not already in the list
	if len(prDates) > 0 && !prDates[len(prDates)-1].Equal(amd) {
		prDates = append(prDates, amd)
	}

	m := len(prDates)
	if m <= 0 {
		return state.NextPrincipalPayment
	}

	// Get parameters from current state
	// Use absolute value of principal - NextPrincipalPayment is always stored positive
	principal := state.NotionalPrincipal.Abs()
	rate := state.NominalInterestRate
	dcc := a.Attributes.DayCountConvention

	// Calculate accrued interest from last IP date to first PR date
	// The accrued interest should be from the last IP that paid interest, not from PRF time
	firstPR := prDates[0]
	lastAccrualDate := eventTime

	// Find the last IP event before the first PR
	// This is important because IP resets accrued interest to zero
	if a.Attributes.CycleAnchorDateOfInterestPayment != nil && a.Attributes.CycleOfInterestPayment != "" {
		ipAnchor := *a.Attributes.CycleAnchorDateOfInterestPayment
		ipCycle, err := utils.ParseCycle(a.Attributes.CycleOfInterestPayment)

		if err == nil && ipCycle != nil {
			// Generate IP dates up to first PR
			ipDates := utils.GenerateCyclicDates(ipAnchor, ipCycle, firstPR, false)
			// Find the LAST IP event that is BEFORE firstPR (not equal)
			// If IP and PR are on the same day, PR happens first (order 3 vs 9)
			// so we need the previous IP date
			for _, ipDate := range ipDates {
				if ipDate.Before(firstPR) {
					lastAccrualDate = ipDate
				}
			}
		}
	}

	// Calculate accrued interest from last accrual date to first PR
	// CRITICAL FIX: Handle rate reset that occurs between lastAccrualDate and firstPR
	// If a rate reset occurred, we need to split the interest calculation:
	// - Period from lastAccrualDate to rrDate: use old rate
	// - Period from rrDate to firstPR: use new rate
	var accruedInt decimal.Decimal

	rrDate := state.LastRateResetDate
	prevRate := state.PreviousNominalInterestRate

	// Check if rate reset occurred between lastAccrualDate and firstPR
	if !rrDate.IsZero() && !prevRate.IsZero() &&
		rrDate.After(lastAccrualDate) && (rrDate.Before(firstPR) || rrDate.Equal(firstPR)) {
		// Split calculation:
		// 1. Interest from lastAccrualDate to rrDate at OLD rate
		yf1 := utils.YearFraction(lastAccrualDate, rrDate, dcc)
		interest1 := principal.Mul(prevRate).Mul(yf1)

		// 2. Interest from rrDate to firstPR at NEW rate
		yf2 := utils.YearFraction(rrDate, firstPR, dcc)
		interest2 := principal.Mul(rate).Mul(yf2)

		accruedInt = interest1.Add(interest2)
	} else {
		// No rate reset in this period, use current rate for entire period
		lastToPRyf := utils.YearFraction(lastAccrualDate, firstPR, dcc)
		accruedInt = principal.Mul(rate).Mul(lastToPRyf)
	}

	// If rate is zero, simple equal payment
	if rate.IsZero() {
		// Safety check: ensure m > 0 to prevent division by zero
		if m <= 0 {
			return decimal.Zero
		}
		return principal.Add(accruedInt).Div(decimal.NewFromInt(int64(m)))
	}

	// ACTUS formula uses year fractions Y(ti, ti+1) between consecutive PR dates
	if m == 1 {
		return principal.Add(accruedInt)
	}

	// Calculate year fractions between consecutive PR dates
	yearFractions := make([]decimal.Decimal, m-1)
	for i := 0; i < m-1; i++ {
		yf := utils.YearFraction(prDates[i], prDates[i+1], dcc)
		yearFractions[i] = yf
	}

	// PRECISION OPTIMIZATION: Increase precision for complex annuity formula
	// This reduces cumulative rounding errors in the nested multiplication/division
	oldPrecision := decimal.DivisionPrecision
	decimal.DivisionPrecision = 32 // Increase from default 16 to 32
	defer func() { decimal.DivisionPrecision = oldPrecision }()

	// ACTUS formula: A(s,T,n,a,r) = (n+a) × ∏[1 + r×Y(ti,ti+1)] / [1 + ∑∏[1 + r×Y(tj,tj+1)]]
	numeratorProduct := decimal.NewFromInt(1)
	for i := 0; i < m-1; i++ {
		term := decimal.NewFromInt(1).Add(rate.Mul(yearFractions[i]))
		numeratorProduct = numeratorProduct.Mul(term)
	}
	numerator := principal.Add(accruedInt).Mul(numeratorProduct)

	denominatorSum := decimal.Zero
	for i := 0; i < m-1; i++ {
		product := decimal.NewFromInt(1)
		for j := i; j < m-1; j++ {
			term := decimal.NewFromInt(1).Add(rate.Mul(yearFractions[j]))
			product = product.Mul(term)
		}
		denominatorSum = denominatorSum.Add(product)
	}
	denominator := decimal.NewFromInt(1).Add(denominatorSum)

	if denominator.IsZero() {
		// Safety check: ensure m > 0 to prevent division by zero
		if m <= 0 {
			return decimal.Zero
		}
		return principal.Add(accruedInt).Div(decimal.NewFromInt(int64(m)))
	}

	// Return positive payment amount (roleSign is applied in POF functions)
	return numerator.Div(denominator)
}

// StateTransition performs state transition for ANN-specific events
func (a *ANN) StateTransition(
	event events.ContractEvent,
	preState *states.ContractState,
	rf riskfactor.Observer,
) (*states.ContractState, error) {
	postState := preState.Clone()

	// Handle ANN-specific events
	switch event.Type {
	case events.IED:
		return a.stfIED(postState, event), nil
	case events.PRF:
		return a.stfPRF(postState, event), nil
	case events.PR:
		return a.stfPR(postState, event), nil
	case events.IP:
		return a.stfIP(postState, event), nil
	case events.IPCB:
		return a.stfIPCB(postState, event), nil
	default:
		// Use PAM's state transition for other events
		return a.PAM.StateTransition(event, preState, rf)
	}
}

// CalculatePayoff calculates the payoff for ANN-specific events
func (a *ANN) CalculatePayoff(
	event events.ContractEvent,
	state *states.ContractState,
	rf riskfactor.Observer,
) (decimal.Decimal, error) {
	switch event.Type {
	case events.PRF:
		return decimal.Zero, nil // PRF has zero payoff
	case events.PR:
		return a.pofPR(state, event.Time), nil
	case events.IP:
		return a.pofIP(state, event.Time), nil
	default:
		// Use PAM's payoff calculation for other events
		return a.PAM.CalculatePayoff(event, state, rf)
	}
}
