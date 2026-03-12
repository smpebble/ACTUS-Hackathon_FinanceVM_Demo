package swaps

import (
	"sort"
	"time"

	"github.com/yourusername/actus-go/pkg/actus/events"
	"github.com/yourusername/actus-go/pkg/actus/utils"
)

// GenerateSchedule generates the event schedule for the SWAPS contract
// SWAPS generates:
// - IED: Initial Exchange Date
// - IP: Interest Payment events (for both fixed and floating legs)
// - RR: Rate Reset events (for floating leg, if applicable)
// - MD: Maturity Date
func (s *SWAPS) GenerateSchedule() (events.EventSchedule, error) {
	schedule := events.EventSchedule{}

	// 1. Add IED (Initial Exchange Date)
	if !s.Attributes.InitialExchangeDate.IsZero() {
		schedule = append(schedule, events.ContractEvent{
			Type:       events.IED,
			Time:       s.Attributes.InitialExchangeDate,
			EventOrder: events.EventSequence[events.IED],
		})
	}

	// 2. Generate IP (Interest Payment) events
	// Both fixed and floating legs have the same payment schedule in this simplified implementation
	ipEvents, err := s.generateIPEvents()
	if err != nil {
		return nil, err
	}
	schedule = append(schedule, ipEvents...)

	// 3. Generate RR (Rate Reset) events for floating leg
	// Rate resets typically occur before each IP event
	if s.Attributes.MarketObjectCodeOfRateReset != "" {
		rrEvents, err := s.generateRREvents()
		if err != nil {
			return nil, err
		}
		schedule = append(schedule, rrEvents...)
	}

	// 4. Add MD (Maturity Date)
	if s.Attributes.MaturityDate != nil && !s.Attributes.MaturityDate.IsZero() {
		schedule = append(schedule, events.ContractEvent{
			Type:       events.MD,
			Time:       *s.Attributes.MaturityDate,
			EventOrder: events.EventSequence[events.MD],
		})
	}

	// Sort events by time and event order
	sort.Slice(schedule, func(i, j int) bool {
		if schedule[i].Time.Equal(schedule[j].Time) {
			return schedule[i].EventOrder < schedule[j].EventOrder
		}
		return schedule[i].Time.Before(schedule[j].Time)
	})

	return schedule, nil
}

// generateIPEvents generates Interest Payment events
func (s *SWAPS) generateIPEvents() (events.EventSchedule, error) {
	schedule := events.EventSchedule{}

	// Parse interest payment cycle
	cycle, err := utils.ParseCycle(s.Attributes.CycleOfInterestPayment)
	if err != nil {
		return nil, err
	}

	// Determine anchor date
	anchorDate := s.Attributes.InitialExchangeDate
	if s.Attributes.CycleAnchorDateOfInterestPayment != nil {
		anchorDate = *s.Attributes.CycleAnchorDateOfInterestPayment
	}

	// Determine end date
	endDate := time.Now().AddDate(100, 0, 0) // Default far future
	if s.Attributes.MaturityDate != nil {
		endDate = *s.Attributes.MaturityDate
	}

	// Generate cyclic dates
	dates := utils.GenerateCyclicDates(anchorDate, cycle, endDate, false)

	// Create IP events
	for _, date := range dates {
		// Skip if date is before status date
		if date.Before(s.Attributes.StatusDate) {
			continue
		}

		// Skip if date is the initial exchange date
		if date.Equal(s.Attributes.InitialExchangeDate) {
			continue
		}

		schedule = append(schedule, events.ContractEvent{
			Type:       events.IP,
			Time:       date,
			EventOrder: events.EventSequence[events.IP],
		})
	}

	return schedule, nil
}

// generateRREvents generates Rate Reset events for floating leg
// Rate resets typically occur a few days before each IP event
func (s *SWAPS) generateRREvents() (events.EventSchedule, error) {
	schedule := events.EventSchedule{}

	// Parse rate reset cycle (typically same as IP cycle for simple swaps)
	var cycle *utils.Cycle
	var err error

	if s.Attributes.CycleOfRateReset != "" {
		cycle, err = utils.ParseCycle(s.Attributes.CycleOfRateReset)
		if err != nil {
			return nil, err
		}
	} else {
		// Default to same cycle as interest payments
		cycle, err = utils.ParseCycle(s.Attributes.CycleOfInterestPayment)
		if err != nil {
			return nil, err
		}
	}

	// Determine anchor date
	anchorDate := s.Attributes.InitialExchangeDate
	if s.Attributes.CycleAnchorDateOfRateReset != nil {
		anchorDate = *s.Attributes.CycleAnchorDateOfRateReset
	}

	// Determine end date
	endDate := time.Now().AddDate(100, 0, 0)
	if s.Attributes.MaturityDate != nil {
		endDate = *s.Attributes.MaturityDate
	}

	// Generate cyclic dates
	dates := utils.GenerateCyclicDates(anchorDate, cycle, endDate, false)

	// Create RR events
	for _, date := range dates {
		// Skip if date is before status date
		if date.Before(s.Attributes.StatusDate) {
			continue
		}

		// Rate reset happens at the beginning of each period (before IP)
		// For simplicity, we place it on the same date as IP but with higher priority
		schedule = append(schedule, events.ContractEvent{
			Type:       events.RR,
			Time:       date,
			EventOrder: events.EventSequence[events.RR],
		})
	}

	return schedule, nil
}
