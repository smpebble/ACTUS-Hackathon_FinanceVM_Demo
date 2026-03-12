package scheduler

import (
	"time"

	"github.com/shopspring/decimal"
	"github.com/yourusername/actus-go/pkg/actus/events"
	"github.com/yourusername/actus-go/pkg/actus/types"
	"github.com/yourusername/actus-go/pkg/actus/utils"
)

// Scheduler generates contract event schedules according to ACTUS specification
type Scheduler struct {
	attrs    *types.ContractAttributes
	calendar utils.Calendar
}

// NewScheduler creates a new event scheduler
func NewScheduler(attrs *types.ContractAttributes) *Scheduler {
	return &Scheduler{
		attrs:    attrs,
		calendar: utils.GetCalendar(attrs.Calendar),
	}
}

// calculateImplicitAmortizationDate calculates when an ANN contract will be fully amortized
// This is used when no MaturityDate or AmortizationDate is specified but NextPrincipalRedemptionPayment is given
func (s *Scheduler) calculateImplicitAmortizationDate() *time.Time {
	// Only applicable for ANN contracts without maturity/amortization date
	if s.attrs.ContractType != "ANN" {
		return nil
	}
	if s.attrs.MaturityDate != nil || s.attrs.AmortizationDate != nil {
		return nil
	}
	if s.attrs.NextPrincipalRedemptionPayment.IsZero() {
		return nil
	}

	// Parse PR cycle
	cycle, err := utils.ParseCycle(s.attrs.CycleOfPrincipalRedemption)
	if err != nil || cycle == nil {
		return nil
	}

	// Get parameters
	principal := s.attrs.NotionalPrincipal
	payment := s.attrs.NextPrincipalRedemptionPayment
	rate := s.attrs.NominalInterestRate
	dcc := s.attrs.DayCountConvention

	// Determine anchor date
	anchor := s.attrs.InitialExchangeDate
	if s.attrs.CycleAnchorDateOfPrincipalRedemption != nil {
		anchor = *s.attrs.CycleAnchorDateOfPrincipalRedemption
	}

	// Simulate amortization to find when principal reaches zero
	// Generate dates for simulation (max 1000 periods to prevent infinite loop)
	maxPeriods := 1000
	farFuture := anchor.AddDate(100, 0, 0)
	dates := utils.GenerateCyclicDates(anchor, cycle, farFuture, false)

	if len(dates) == 0 {
		return nil
	}

	currentPrincipal := principal
	lastDate := dates[0]

	for i := 0; i < len(dates) && i < maxPeriods; i++ {
		currentDate := dates[i]
		if currentDate.Before(s.attrs.InitialExchangeDate) || currentDate.Equal(s.attrs.InitialExchangeDate) {
			lastDate = currentDate
			continue
		}

		// Calculate interest for this period
		yf := utils.YearFraction(lastDate, currentDate, dcc)
		interest := currentPrincipal.Mul(rate).Mul(yf)

		// Principal portion = Payment - Interest
		principalPortion := payment.Sub(interest)

		// Check if this is the last period
		if principalPortion.GreaterThanOrEqual(currentPrincipal) {
			// This is the final period - principal will be fully paid
			result := currentDate
			return &result
		}

		// Update principal for next iteration
		currentPrincipal = currentPrincipal.Sub(principalPortion)
		if currentPrincipal.LessThanOrEqual(decimal.Zero) {
			result := currentDate
			return &result
		}

		lastDate = currentDate
	}

	return nil
}

// Schedule generates the complete event schedule for the contract
func (s *Scheduler) Schedule() (events.EventSchedule, error) {
	schedule := events.EventSchedule{}

	// Handle Status Date and IED based on their relationship
	if !s.attrs.StatusDate.IsZero() {
		if s.attrs.StatusDate.Before(s.attrs.InitialExchangeDate) {
			// 1. AD before IED: Generate AD event, then IED
			schedule = append(schedule, events.NewContractEvent(
				events.AD,
				s.attrs.StatusDate,
				s.attrs.Currency,
			))
			schedule = append(schedule, events.NewContractEvent(
				events.IED,
				s.attrs.InitialExchangeDate,
				s.attrs.Currency,
			))
		} else if s.attrs.StatusDate.Equal(s.attrs.InitialExchangeDate) {
			// 2. StatusDate equals IED: Only generate IED
			schedule = append(schedule, events.NewContractEvent(
				events.IED,
				s.attrs.InitialExchangeDate,
				s.attrs.Currency,
			))
		}
		// 3. StatusDate after IED: Don't generate IED or AD
		// InitializeState already set up the state at StatusDate
	} else {
		// No StatusDate: Generate IED normally
		schedule = append(schedule, events.NewContractEvent(
			events.IED,
			s.attrs.InitialExchangeDate,
			s.attrs.Currency,
		))
	}

	// 4. IP (Interest Payment) and IPCI (Interest Capitalization) - if interest rate is specified
	if !s.attrs.NominalInterestRate.IsZero() && s.attrs.CycleOfInterestPayment != "" {
		ipSchedule := s.generateIPSchedule()

		// Convert IP to IPCI events if capitalization end date is specified
		if s.attrs.CapitalizationEndDate != nil {
			// Check if capitalization end date matches any existing IP event
			hasEventAtCapEnd := false
			for i := range ipSchedule {
				// If IP event is at or before capitalization end date, convert to IPCI
				if ipSchedule[i].Time.Before(*s.attrs.CapitalizationEndDate) ||
					ipSchedule[i].Time.Equal(*s.attrs.CapitalizationEndDate) {
					ipSchedule[i].Type = events.IPCI
					if ipSchedule[i].Time.Equal(*s.attrs.CapitalizationEndDate) {
						hasEventAtCapEnd = true
					}
				}
			}

			// If capitalization end date doesn't match any IP event, add a final IPCI event
			if !hasEventAtCapEnd && s.attrs.CapitalizationEndDate.After(s.attrs.InitialExchangeDate) {
				finalIPCI := events.NewContractEvent(
					events.IPCI,
					*s.attrs.CapitalizationEndDate,
					s.attrs.Currency,
				)
				finalIPCI.ScheduleTime = *s.attrs.CapitalizationEndDate
				ipSchedule = append(ipSchedule, finalIPCI)
			}
		}

		schedule = append(schedule, ipSchedule...)
	}

	// 4. PR (Principal Redemption) - for LAM/ANN contracts
	if s.attrs.CycleOfPrincipalRedemption != "" {
		prSchedule := s.generatePRSchedule()
		schedule = append(schedule, prSchedule...)
	}

	// 5. RR (Rate Reset) - for variable rate contracts
	if s.attrs.MarketObjectCodeOfRateReset != "" && s.attrs.CycleOfRateReset != "" {
		rrSchedule := s.generateRRSchedule()
		schedule = append(schedule, rrSchedule...)
	}

	// 6. FP (Fee Payment) - if fees are specified
	if !s.attrs.FeeRate.IsZero() && s.attrs.CycleOfFee != "" {
		fpSchedule := s.generateFPSchedule()
		schedule = append(schedule, fpSchedule...)
	}

	// 6.5 IPCB (Interest Calculation Base) - for LAM/ANN with IPCB cycle
	ipcbSchedule := s.generateIPCBSchedule()
	schedule = append(schedule, ipcbSchedule...)

	// 6.6 SC (Scaling Index) - if scaling index is specified
	if s.attrs.MarketObjectCodeOfScalingIndex != "" && s.attrs.CycleOfScalingIndex != "" {
		scSchedule := s.generateSCSchedule()
		schedule = append(schedule, scSchedule...)
	}

	// 7. PRD (Purchase) - if purchase date is specified
	if s.attrs.PurchaseDate != nil {
		schedule = append(schedule, events.NewContractEvent(
			events.PRD,
			*s.attrs.PurchaseDate,
			s.attrs.Currency,
		))
	}

	// 8. TD (Termination) - if termination date is specified
	if s.attrs.TerminationDate != nil {
		schedule = append(schedule, events.NewContractEvent(
			events.TD,
			*s.attrs.TerminationDate,
			s.attrs.Currency,
		))
	}

	// 9. MD (Maturity) - if maturity date is specified or calculated for ANN
	maturityDate := s.attrs.MaturityDate
	// For ANN contracts without maturity date, calculate implicit amortization date
	if maturityDate == nil && s.attrs.ContractType == "ANN" {
		maturityDate = s.calculateImplicitAmortizationDate()
	}
	if maturityDate != nil {
		// Apply business day convention to maturity date
		adjustedMD := s.applyBDConvention([]time.Time{*maturityDate})
		mdTime := *maturityDate
		if len(adjustedMD) > 0 {
			mdTime = adjustedMD[0]
		}
		md := events.NewContractEvent(
			events.MD,
			mdTime,
			s.attrs.Currency,
		)
		// Ensure ScheduleTime is set for MD event
		md.ScheduleTime = *maturityDate
		schedule = append(schedule, md)
	}

	// Sort events by time and priority
	schedule.Sort()

	return schedule, nil
}

// generateIPSchedule generates Interest Payment events
func (s *Scheduler) generateIPSchedule() events.EventSchedule {
	cycle, err := utils.ParseCycle(s.attrs.CycleOfInterestPayment)
	if err != nil || cycle == nil {
		return events.EventSchedule{}
	}

	// Determine anchor date
	anchor := s.attrs.InitialExchangeDate
	if s.attrs.CycleAnchorDateOfInterestPayment != nil {
		anchor = *s.attrs.CycleAnchorDateOfInterestPayment
	}

	// Determine end date for cycle generation (use MaturityDate, not TerminationDate)
	// TerminationDate will be used to filter events later
	end := time.Now().AddDate(100, 0, 0) // Default: 100 years
	if s.attrs.MaturityDate != nil {
		end = *s.attrs.MaturityDate
	} else if s.attrs.ContractType == "ANN" {
		// For ANN without maturity, calculate implicit amortization date
		if implicitEnd := s.calculateImplicitAmortizationDate(); implicitEnd != nil {
			end = *implicitEnd
		}
	}

	// Generate cyclic dates
	dates := utils.GenerateCyclicDates(anchor, cycle, end, false)

	// Apply End-of-Month convention (only for month-based periods)
	// Day and Week based periods should not be adjusted by EOM convention
	if cycle.Period == "M" || cycle.Period == "Q" || cycle.Period == "H" || cycle.Period == "Y" {
		// Save the last date if it's a long stub (maturity date)
		var lastDate *time.Time
		if cycle.Stub == "+" && len(dates) > 0 && dates[len(dates)-1].Equal(end) {
			lastDate = &dates[len(dates)-1]
			dates = dates[:len(dates)-1] // Temporarily remove it
		}

		dates = s.applyEOMConvention(dates, anchor)

		// Restore the long stub end date (it should not be adjusted by EOM)
		if lastDate != nil {
			dates = append(dates, *lastDate)
		}
	}

	// Store schedule dates (before business day adjustment)
	scheduleDates := make([]time.Time, len(dates))
	copy(scheduleDates, dates)

	// Apply Business Day convention to get payment dates
	dates = s.applyBDConvention(dates)

	// Convert to events
	// Include dates after IED (or at IED if anchor equals IED)
	// Exclude dates at or after TerminationDate (if specified)
	schedule := events.EventSchedule{}
	for i, date := range dates {
		// Skip if at or after termination date
		if s.attrs.TerminationDate != nil && !date.Before(*s.attrs.TerminationDate) {
			continue
		}

		// Include if date is after IED
		if date.After(s.attrs.InitialExchangeDate) {
			event := events.NewContractEvent(
				events.IP,
				date, // Payment date (adjusted)
				s.attrs.Currency,
			)
			event.ScheduleTime = scheduleDates[i] // Schedule date (unadjusted)
			schedule = append(schedule, event)
		} else if date.Equal(s.attrs.InitialExchangeDate) && anchor.Equal(s.attrs.InitialExchangeDate) {
			// Also include if date equals IED and anchor equals IED
			// This represents the first IP event at the start of the cycle
			event := events.NewContractEvent(
				events.IP,
				date, // Payment date (adjusted)
				s.attrs.Currency,
			)
			event.ScheduleTime = scheduleDates[i] // Schedule date (unadjusted)
			schedule = append(schedule, event)
		}
	}

	// ACTUS requires a final IP event at maturity date if not already included
	// This ensures final accrued interest is paid
	// For ANN without maturity, use implicit amortization date
	effectiveMaturityDate := s.attrs.MaturityDate
	if effectiveMaturityDate == nil && s.attrs.ContractType == "ANN" {
		effectiveMaturityDate = s.calculateImplicitAmortizationDate()
	}
	if effectiveMaturityDate != nil {
		// Apply business day convention to maturity date
		adjustedDates := s.applyBDConvention([]time.Time{*effectiveMaturityDate})
		adjustedMD := *effectiveMaturityDate
		if len(adjustedDates) > 0 {
			adjustedMD = adjustedDates[0]
		}

		mdExists := false
		for _, event := range schedule {
			if event.Time.Equal(adjustedMD) {
				mdExists = true
				break
			}
		}

		// Add IP at maturity date if:
		// 1. Not already in schedule
		// 2. Maturity date is after IED
		if !mdExists && effectiveMaturityDate.After(s.attrs.InitialExchangeDate) {
			event := events.NewContractEvent(
				events.IP,
				adjustedMD,
				s.attrs.Currency,
			)
			// For maturity date IP, schedule time = payment time (no adjustment needed)
			event.ScheduleTime = *effectiveMaturityDate
			schedule = append(schedule, event)
		}
	}

	return schedule
}

// generatePRSchedule generates Principal Redemption events
func (s *Scheduler) generatePRSchedule() events.EventSchedule {
	cycle, err := utils.ParseCycle(s.attrs.CycleOfPrincipalRedemption)
	if err != nil || cycle == nil {
		return events.EventSchedule{}
	}

	// Determine anchor date
	anchor := s.attrs.InitialExchangeDate
	if s.attrs.CycleAnchorDateOfPrincipalRedemption != nil {
		anchor = *s.attrs.CycleAnchorDateOfPrincipalRedemption
	}

	// Determine end date for cycle generation (use MaturityDate, not TerminationDate)
	// TerminationDate will be used to filter events later
	end := time.Now().AddDate(100, 0, 0)
	if s.attrs.MaturityDate != nil {
		end = *s.attrs.MaturityDate
	} else if s.attrs.ContractType == "ANN" {
		// For ANN without maturity, calculate implicit amortization date
		if implicitEnd := s.calculateImplicitAmortizationDate(); implicitEnd != nil {
			end = *implicitEnd
		}
	}

	// Generate cyclic dates
	dates := utils.GenerateCyclicDates(anchor, cycle, end, false)

	// Apply End-of-Month convention (only for month-based periods)
	// Day and Week based periods should not be adjusted by EOM convention
	if cycle.Period == "M" || cycle.Period == "Q" || cycle.Period == "H" || cycle.Period == "Y" {
		// Save the last date if it's a long stub (maturity date)
		var lastDate *time.Time
		if cycle.Stub == "+" && len(dates) > 0 && dates[len(dates)-1].Equal(end) {
			lastDate = &dates[len(dates)-1]
			dates = dates[:len(dates)-1] // Temporarily remove it
		}

		dates = s.applyEOMConvention(dates, anchor)

		// Restore the long stub end date (it should not be adjusted by EOM)
		if lastDate != nil {
			dates = append(dates, *lastDate)
		}
	}

	// Store schedule dates (before business day adjustment)
	scheduleDates := make([]time.Time, len(dates))
	copy(scheduleDates, dates)

	// Apply Business Day convention
	dates = s.applyBDConvention(dates)

	// Convert to events (exclude dates before IED, at maturity, or at/after termination)
	schedule := events.EventSchedule{}
	initialScheduleSize := 0

	// Determine effective maturity date for ANN contracts
	effectiveMaturityForPR := s.attrs.MaturityDate
	if effectiveMaturityForPR == nil && s.attrs.ContractType == "ANN" {
		effectiveMaturityForPR = s.calculateImplicitAmortizationDate()
	}

	for i, date := range dates {
		// Exclude PR events before IED (but allow PR on IED itself)
		if date.Before(s.attrs.InitialExchangeDate) {
			continue
		}

		// Exclude PR events at maturity date - remaining principal is paid by MD event
		// Also check the schedule date (before business day adjustment)
		if effectiveMaturityForPR != nil {
			if date.Equal(*effectiveMaturityForPR) || scheduleDates[i].Equal(*effectiveMaturityForPR) {
				continue
			}
		}

		// Exclude PR events at or after termination date
		if s.attrs.TerminationDate != nil && !date.Before(*s.attrs.TerminationDate) {
			continue
		}

		// Generate PRF event one day before PR (if PR is after IED)
		// PRF = Principal Redemption Fixing event
		// This event updates accrued interest but has zero payoff
		if date.After(s.attrs.InitialExchangeDate) {
			prfDate := scheduleDates[i].AddDate(0, 0, -1) // One day before schedule date
			prfEvent := events.NewContractEvent(
				events.PRF,
				prfDate,
				s.attrs.Currency,
			)
			prfEvent.ScheduleTime = prfDate
			schedule = append(schedule, prfEvent)
		}

		// Generate PR event
		event := events.NewContractEvent(
			events.PR,
			date, // Payment date (adjusted)
			s.attrs.Currency,
		)
		event.ScheduleTime = scheduleDates[i] // Schedule date (unadjusted)
		schedule = append(schedule, event)
		initialScheduleSize++
	}

	return schedule
}

// generateRRSchedule generates Rate Reset events
func (s *Scheduler) generateRRSchedule() events.EventSchedule {
	cycle, err := utils.ParseCycle(s.attrs.CycleOfRateReset)
	if err != nil || cycle == nil {
		return events.EventSchedule{}
	}

	// Determine anchor date
	anchor := s.attrs.InitialExchangeDate
	if s.attrs.CycleAnchorDateOfRateReset != nil {
		anchor = *s.attrs.CycleAnchorDateOfRateReset
	}

	// Determine end date for cycle generation (use MaturityDate, not TerminationDate)
	// TerminationDate will be used to filter events later
	// Add extra buffer to ensure all RR events before maturity are generated
	end := time.Now().AddDate(100, 0, 0)
	if s.attrs.MaturityDate != nil {
		// Add one extra cycle period to maturity to handle long stub cases
		end = s.attrs.MaturityDate.AddDate(0, 6, 0) // Add 6 months buffer
	}

	// Generate cyclic dates
	dates := utils.GenerateCyclicDates(anchor, cycle, end, false)

	// Apply End-of-Month convention for RR (same as PR/IP)
	// This is necessary when anchor is at end of month
	if cycle.Period == "M" || cycle.Period == "Q" || cycle.Period == "H" || cycle.Period == "Y" {
		dates = s.applyEOMConvention(dates, anchor)
	}

	// Store schedule dates (before business day adjustment)
	scheduleDates := make([]time.Time, len(dates))
	copy(scheduleDates, dates)

	// Apply Business Day convention
	dates = s.applyBDConvention(dates)

	// Convert to events (exclude at or before IED, and at or after maturity date)
	schedule := events.EventSchedule{}
	firstRREvent := true // Track if this is the first RR event

	for i, date := range dates {
		// Exclude RR events at or before IED
		if !date.After(s.attrs.InitialExchangeDate) {
			continue
		}

		// Exclude RR events at or after maturity date (allow events strictly before maturity)
		if s.attrs.MaturityDate != nil && !date.Before(*s.attrs.MaturityDate) {
			continue
		}

		// Determine event type: RRF for first event if NextResetRate is specified, RR otherwise
		eventType := events.RR
		if firstRREvent && !s.attrs.NextResetRate.IsZero() {
			eventType = events.RRF
			firstRREvent = false
		}

		event := events.NewContractEvent(
			eventType,
			date, // Payment date (adjusted)
			s.attrs.Currency,
		)
		event.ScheduleTime = scheduleDates[i] // Schedule date (unadjusted)
		schedule = append(schedule, event)

		// For ANN contracts, generate a PRF event after each RR event
		// This is because annuity payment needs to be recalculated with the new rate
		// Use PRFAfterRR priority to ensure it executes AFTER the RR event
		if s.attrs.ContractType == "ANN" {
			prfEvent := events.NewContractEvent(
				events.PRF,
				date,
				s.attrs.Currency,
			)
			prfEvent.ScheduleTime = scheduleDates[i]
			prfEvent.EventOrder = events.PRFAfterRR // Execute after RR
			schedule = append(schedule, prfEvent)
		}
	}

	return schedule
}

// generateFPSchedule generates Fee Payment events
func (s *Scheduler) generateFPSchedule() events.EventSchedule {
	cycle, err := utils.ParseCycle(s.attrs.CycleOfFee)
	if err != nil || cycle == nil {
		return events.EventSchedule{}
	}

	// Determine anchor date
	anchor := s.attrs.InitialExchangeDate
	if s.attrs.CycleAnchorDateOfFee != nil {
		anchor = *s.attrs.CycleAnchorDateOfFee
	}

	// Determine end date for cycle generation (use MaturityDate, not TerminationDate)
	// TerminationDate will be used to filter events later
	end := time.Now().AddDate(100, 0, 0)
	if s.attrs.MaturityDate != nil {
		end = *s.attrs.MaturityDate
	}

	// Generate cyclic dates
	dates := utils.GenerateCyclicDates(anchor, cycle, end, false)

	// Store schedule dates (before business day adjustment)
	scheduleDates := make([]time.Time, len(dates))
	copy(scheduleDates, dates)

	// Apply Business Day convention
	dates = s.applyBDConvention(dates)

	// Convert to events
	schedule := events.EventSchedule{}
	for i, date := range dates {
		if date.After(s.attrs.InitialExchangeDate) {
			event := events.NewContractEvent(
				events.FP,
				date, // Payment date (adjusted)
				s.attrs.Currency,
			)
			event.ScheduleTime = scheduleDates[i] // Schedule date (unadjusted)
			schedule = append(schedule, event)
		}
	}

	return schedule
}

// generateSCSchedule generates Scaling Index Fixing events
func (s *Scheduler) generateSCSchedule() events.EventSchedule {
	cycle, err := utils.ParseCycle(s.attrs.CycleOfScalingIndex)
	if err != nil || cycle == nil {
		return events.EventSchedule{}
	}

	// Determine anchor date
	anchor := s.attrs.InitialExchangeDate
	if s.attrs.CycleAnchorDateOfScalingIndex != nil {
		anchor = *s.attrs.CycleAnchorDateOfScalingIndex
	}

	// Determine end date (use earlier of MaturityDate and TerminationDate)
	end := time.Now().AddDate(100, 0, 0)
	if s.attrs.MaturityDate != nil {
		end = *s.attrs.MaturityDate
	}
	if s.attrs.TerminationDate != nil && s.attrs.TerminationDate.Before(end) {
		end = *s.attrs.TerminationDate
	}
	if s.attrs.TerminationDate != nil {
		end = *s.attrs.TerminationDate
	}

	// Generate cyclic dates
	dates := utils.GenerateCyclicDates(anchor, cycle, end, false)

	// Store schedule dates (before business day adjustment)
	scheduleDates := make([]time.Time, len(dates))
	copy(scheduleDates, dates)

	// Apply Business Day convention
	dates = s.applyBDConvention(dates)

	// Convert to events (exclude at or before IED, and at maturity date)
	schedule := events.EventSchedule{}
	for i, date := range dates {
		// Exclude SC events at or before IED
		if !date.After(s.attrs.InitialExchangeDate) {
			continue
		}

		// Exclude SC events at maturity date
		if s.attrs.MaturityDate != nil && date.Equal(*s.attrs.MaturityDate) {
			continue
		}

		event := events.NewContractEvent(
			events.SC,
			date, // Payment date (adjusted)
			s.attrs.Currency,
		)
		event.ScheduleTime = scheduleDates[i] // Schedule date (unadjusted)
		schedule = append(schedule, event)
	}

	return schedule
}

// generateIPCBSchedule generates Interest Calculation Base Fixing events for LAM/ANN
func (s *Scheduler) generateIPCBSchedule() events.EventSchedule {
	// IPCB only relevant for LAM/ANN contracts
	if s.attrs.ContractType != "LAM" && s.attrs.ContractType != "ANN" {
		return events.EventSchedule{}
	}

	// IPCB only needed when there's a cycle specified
	if s.attrs.CycleOfInterestCalculationBase == "" {
		return events.EventSchedule{}
	}

	cycle, err := utils.ParseCycle(s.attrs.CycleOfInterestCalculationBase)
	if err != nil || cycle == nil {
		return events.EventSchedule{}
	}

	// Determine anchor date
	anchor := s.attrs.InitialExchangeDate
	if s.attrs.CycleAnchorDateOfInterestCalculationBase != nil {
		anchor = *s.attrs.CycleAnchorDateOfInterestCalculationBase
	}

	// Determine end date for cycle generation (use MaturityDate, not TerminationDate)
	// TerminationDate will be used to filter events later
	end := time.Now().AddDate(100, 0, 0)
	if s.attrs.MaturityDate != nil {
		end = *s.attrs.MaturityDate
	}

	// Generate cyclic dates
	dates := utils.GenerateCyclicDates(anchor, cycle, end, false)

	// Store schedule dates (before business day adjustment)
	scheduleDates := make([]time.Time, len(dates))
	copy(scheduleDates, dates)

	// Apply Business Day convention
	dates = s.applyBDConvention(dates)

	// Convert to events (exclude at or before IED, and at maturity date)
	schedule := events.EventSchedule{}
	for i, date := range dates {
		// Exclude IPCB events at or before IED
		if !date.After(s.attrs.InitialExchangeDate) {
			continue
		}

		// Exclude IPCB events at maturity date
		if s.attrs.MaturityDate != nil && date.Equal(*s.attrs.MaturityDate) {
			continue
		}

		event := events.NewContractEvent(
			events.IPCB,
			date, // Payment date (adjusted)
			s.attrs.Currency,
		)
		event.ScheduleTime = scheduleDates[i] // Schedule date (unadjusted)
		schedule = append(schedule, event)
	}

	return schedule
}

// applyEOMConvention applies End-of-Month convention to dates
func (s *Scheduler) applyEOMConvention(dates []time.Time, anchor time.Time) []time.Time {
	anchorDay := anchor.Day()

	switch s.attrs.EndOfMonthConvention {
	case types.EOMC_EOM:
		// EOM: Check if anchor is on the last day of its month
		// If yes, move all dates to end of month
		// If no, use same day convention (keep anchor day when possible)
		anchorLastDay := time.Date(anchor.Year(), anchor.Month()+1, 0, 0, 0, 0, 0, time.UTC).Day()
		if anchorDay == anchorLastDay {
			// Anchor is on last day of month, so all dates should be on last day
			return s.adjustToEndOfMonth(dates)
		} else {
			// Anchor is not on last day, use same day convention
			return s.adjustToSameDay(dates, anchorDay)
		}

	case types.EOMC_SD:
		// SD (Same Day): Always try to keep the same day of month as anchor
		// This is the default ACTUS behavior
		return s.adjustToSameDay(dates, anchorDay)

	default:
		// No convention or NULL: return as is
		return dates
	}
}

// adjustToSameDay adjusts dates to maintain the same day of month as the anchor
// For months that don't have enough days, use the last day of that month
func (s *Scheduler) adjustToSameDay(dates []time.Time, targetDay int) []time.Time {
	adjusted := make([]time.Time, len(dates))
	for i, date := range dates {
		// Get the last day of this month
		lastDay := time.Date(date.Year(), date.Month()+1, 0, 0, 0, 0, 0, time.UTC).Day()

		// Use target day if the month has enough days, otherwise use last day
		actualDay := targetDay
		if targetDay > lastDay {
			actualDay = lastDay
		}

		adjusted[i] = time.Date(
			date.Year(),
			date.Month(),
			actualDay,
			date.Hour(),
			date.Minute(),
			date.Second(),
			date.Nanosecond(),
			date.Location(),
		)
	}
	return adjusted
}

// adjustToEndOfMonth adjusts all dates to the last day of their respective months
func (s *Scheduler) adjustToEndOfMonth(dates []time.Time) []time.Time {
	adjusted := make([]time.Time, len(dates))
	for i, date := range dates {
		// Get last day of the month
		lastDay := time.Date(date.Year(), date.Month()+1, 0,
			date.Hour(), date.Minute(), date.Second(),
			date.Nanosecond(), date.Location())
		adjusted[i] = lastDay
	}
	return adjusted
}

// applyBDConvention applies Business Day Convention to dates
func (s *Scheduler) applyBDConvention(dates []time.Time) []time.Time {
	if s.attrs.BusinessDayConvention == types.BDC_NULL {
		return dates
	}

	adjusted := make([]time.Time, len(dates))
	for i, date := range dates {
		adjusted[i] = s.calendar.AdjustDate(date, s.attrs.BusinessDayConvention)
	}

	return adjusted
}
