package pam

import (
	"time"

	"github.com/shopspring/decimal"
	"github.com/yourusername/actus-go/pkg/actus/states"
	"github.com/yourusername/actus-go/pkg/actus/utils"
)

// Payoff Functions (POF) for PAM contract
// These implement the ACTUS specification section 4.x for PAM contracts

// pofAD implements the Analysis Date payoff
// No cashflow at analysis date
func (p *PAM) pofAD() decimal.Decimal {
	return decimal.Zero
}

// pofIED implements the Initial Exchange Date payoff
// Formula: POF_IED_PAM = R(CNTRL) × (-1) × (NT + PDIED)
// This represents the initial disbursement of the loan
func (p *PAM) pofIED(state *states.ContractState) decimal.Decimal {
	roleSign := p.getRoleSign()

	// Payoff = sign × (-1) × (notional + premium/discount)
	// Negative sign because disbursement is cash outflow for lender
	payoff := roleSign.Mul(decimal.NewFromInt(-1)).Mul(
		p.Attributes.NotionalPrincipal.Add(p.Attributes.PremiumDiscountAtIED),
	)

	return payoff
}

// pofIP implements the Interest Payment payoff
// Formula: POF_IP_PAM = Isc- × (Ipac- + Y(Sd-, t) × Ipnr- × Nt-)
// This calculates the interest payment including accrued interest
func (p *PAM) pofIP(state *states.ContractState, scheduleTime time.Time, paymentTime time.Time) decimal.Decimal {
	// Determine which date to use for day count calculation based on BDC
	// SCMF/SCF/SCMP/SCP: Shift/Calculate - use payment time (after BD adjustment)
	// CSMF/CSF/CSMP/CSP/NULL: Calculate/Shift - use schedule time (before BD adjustment)
	var calculationTime time.Time
	switch p.Attributes.BusinessDayConvention {
	case "SCMF", "SCF", "SCMP", "SCP":
		calculationTime = paymentTime // Use adjusted date
	default: // CSMF, CSF, CSMP, CSP, NULL
		calculationTime = scheduleTime // Use schedule date
	}

	// Calculate year fraction
	yf := utils.YearFraction(state.StatusDate, calculationTime, p.Attributes.DayCountConvention)

	// Interest payment = scaling index × (accrued interest + period interest)
	// Period interest = year fraction × interest rate × notional
	interestPayment := state.InterestScalingIndex.Mul(
		state.AccruedInterest.Add(
			yf.Mul(state.NominalInterestRate).Mul(state.NotionalPrincipal),
		),
	)

	return interestPayment
}

// pofMD implements the Maturity Date payoff
// Formula: POF_MD_PAM = Nsc- × Nt- + Isc- × Ipac- + Feac-
// This is the final payment including principal, accrued interest, and fees
func (p *PAM) pofMD(state *states.ContractState) decimal.Decimal {
	// Final payment = scaled principal + scaled accrued interest + fees
	payoff := state.NominalScalingIndex.Mul(state.NotionalPrincipal).Add(
		state.InterestScalingIndex.Mul(state.AccruedInterest),
	).Add(state.FeeAccrued)

	return payoff
}

// pofFP implements the Fee Payment payoff
// Calculates fee payment based on fee basis (absolute or notional)
func (p *PAM) pofFP(state *states.ContractState, eventTime time.Time) decimal.Decimal {
	roleSign := p.getRoleSign()

	if p.Attributes.FeeBasis == "A" {
		// Absolute fee: fixed amount
		payoff := roleSign.Mul(p.Attributes.FeeRate)
		return payoff
	}

	// Notional-based fee: calculated on notional principal
	yf := utils.YearFraction(state.StatusDate, eventTime, p.Attributes.DayCountConvention)
	payoff := roleSign.Mul(
		state.FeeAccrued.Add(
			yf.Mul(state.NotionalPrincipal).Mul(p.Attributes.FeeRate),
		),
	)

	return payoff
}

// pofPRD implements the Purchase Date payoff
// Formula: POF_PRD_PAM = R(CNTRL) × (-1) × (PPRD + IPAC + Y(SD, t) × IPNR × NT)
// Buyer pays the purchase price plus accrued interest
func (p *PAM) pofPRD(state *states.ContractState, eventTime time.Time) decimal.Decimal {
	roleSign := p.getRoleSign()

	// Calculate year fraction from last status date to purchase date
	yf := utils.YearFraction(state.StatusDate, eventTime, p.Attributes.DayCountConvention)

	// Calculate accrued interest up to purchase date
	accruedInterest := state.AccruedInterest.Add(
		yf.Mul(state.NominalInterestRate).Mul(state.NotionalPrincipal),
	)

	// Payoff = sign × (-1) × (purchase price + accrued interest)
	// Negative because buyer pays out cash
	payoff := roleSign.Mul(decimal.NewFromInt(-1)).Mul(
		p.Attributes.PriceAtPurchaseDate.Add(accruedInterest),
	)

	return payoff
}

// pofTD implements the Termination Date payoff
// Formula: POF_TD_PAM = R(CNTRL) × (PTD + Ipac + Y(Sd-, t) × Ipnr × Nt)
// Contract is terminated with termination price plus accrued interest
func (p *PAM) pofTD(state *states.ContractState, eventTime time.Time) decimal.Decimal {
	roleSign := p.getRoleSign()

	// Calculate year fraction from last status date to termination date
	yf := utils.YearFraction(state.StatusDate, eventTime, p.Attributes.DayCountConvention)

	// Calculate accrued interest up to termination date
	accruedInterest := state.AccruedInterest.Add(
		yf.Mul(state.NominalInterestRate).Mul(state.NotionalPrincipal),
	)

	// Payoff = sign × (termination price + accrued interest)
	payoff := roleSign.Mul(
		p.Attributes.PriceAtTerminationDate.Add(accruedInterest),
	)

	return payoff
}

// pofSC implements the Scaling Index Fixing payoff
// SC is a state update event with no cashflow
// Formula: POF_SC_PAM = 0
func (p *PAM) pofSC() decimal.Decimal {
	return decimal.Zero
}
