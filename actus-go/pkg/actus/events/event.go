package events

import (
	"sort"
	"time"

	"github.com/shopspring/decimal"
)

// EventType represents the type of contract event according to ACTUS specification
type EventType string

const (
	AD   EventType = "AD"   // Analysis Date
	IED  EventType = "IED"  // Initial Exchange
	FP   EventType = "FP"   // Fee Payment
	PR   EventType = "PR"   // Principal Redemption
	PRF  EventType = "PRF"  // Principal Redemption Fixing
	PI   EventType = "PI"   // Principal Increase (for LAX)
	PP   EventType = "PP"   // Principal Prepayment
	PY   EventType = "PY"   // Penalty Payment
	PRD  EventType = "PRD"  // Purchase
	TD   EventType = "TD"   // Termination
	IP   EventType = "IP"   // Interest Payment
	IPFX EventType = "IPFX" // Interest Payment Fixed Leg (for SWPPV)
	IPFL EventType = "IPFL" // Interest Payment Floating Leg (for SWPPV)
	IPCI EventType = "IPCI" // Interest Capitalization
	IPCB EventType = "IPCB" // Interest Calculation Base Fixing
	RR   EventType = "RR"   // Rate Reset (variable rate)
	RRF  EventType = "RRF"  // Rate Reset Fixed Leg
	SC   EventType = "SC"   // Scaling Index Fixing
	DV   EventType = "DV"   // Dividend Payment
	XD   EventType = "XD"   // Exercise Date (for options)
	STD  EventType = "STD"  // Settlement Date
	MD   EventType = "MD"   // Maturity
	CE   EventType = "CE"   // Credit Event
)

// EventSequence defines the priority order for events occurring at the same time
// Lower numbers execute first
var EventSequence = map[EventType]int{
	AD:   0,
	IED:  1,
	PRF:  2, // Principal redemption fixing before PR
	PR:   3, // Principal redemption
	PI:   3,
	PP:   4,
	PY:   5,
	FP:   6,
	PRD:  7,
	IPFX: 8,  // Interest payment fixed leg (SWPPV) - executes BEFORE IPFL
	IP:   9,  // Interest payment uses CURRENT rate/base
	IPFL: 9,  // Interest payment floating leg (SWPPV) - executes AFTER IPFX
	IPCI: 10, // Interest capitalization after IP
	RR:   11, // Rate reset AFTER IP - updates rate for NEXT period
	RRF:  11,
	IPCB: 12, // IPCB AFTER IP - updates calculation base for NEXT period
	TD:   13, // Termination AFTER all regular payment events (PR, IP, RR, IPCB)
	SC:   14, // Scaling index AFTER IP
	DV:   15,
	XD:   16,
	STD:  17,
	MD:   18,
	CE:   99,
}

// PRFAfterRR is the priority for PRF events that occur after Rate Reset
// These PRF events recalculate the annuity payment with the new rate
const PRFAfterRR = 12

// ContractEvent represents a single event in a contract's lifecycle
type ContractEvent struct {
	Type         EventType       `json:"type"`
	Time         time.Time       `json:"time"`         // Payment date (after business day adjustment)
	ScheduleTime time.Time       `json:"scheduleTime"` // Schedule date (before business day adjustment, used for day count)
	Payoff       decimal.Decimal `json:"payoff"`
	Currency     string          `json:"currency"`
	NominalValue decimal.Decimal `json:"nominalValue,omitempty"` // For display purposes
	EventOrder   int             `json:"eventOrder"`

	// Internal fields (not serialized)
	StateChange string `json:"-"` // Description of state changes
}

// EventSchedule is a collection of contract events
type EventSchedule []ContractEvent

// Sort sorts the event schedule by time and priority
func (es EventSchedule) Sort() {
	sort.Slice(es, func(i, j int) bool {
		// First, sort by time
		if !es[i].Time.Equal(es[j].Time) {
			return es[i].Time.Before(es[j].Time)
		}
		// If times are equal, sort by event order (priority)
		return es[i].EventOrder < es[j].EventOrder
	})
}

// Filter returns events within the specified time range (inclusive)
func (es EventSchedule) Filter(from, to time.Time) EventSchedule {
	filtered := EventSchedule{}
	for _, event := range es {
		if (event.Time.Equal(from) || event.Time.After(from)) &&
			(event.Time.Before(to) || event.Time.Equal(to)) {
			filtered = append(filtered, event)
		}
	}
	return filtered
}

// FilterByType returns all events of the specified type
func (es EventSchedule) FilterByType(eventType EventType) EventSchedule {
	filtered := EventSchedule{}
	for _, event := range es {
		if event.Type == eventType {
			filtered = append(filtered, event)
		}
	}
	return filtered
}

// Count returns the number of events
func (es EventSchedule) Count() int {
	return len(es)
}

// TotalPayoff calculates the sum of all payoffs
func (es EventSchedule) TotalPayoff() decimal.Decimal {
	total := decimal.Zero
	for _, event := range es {
		total = total.Add(event.Payoff)
	}
	return total
}

// First returns the first event in the schedule
func (es EventSchedule) First() *ContractEvent {
	if len(es) == 0 {
		return nil
	}
	return &es[0]
}

// Last returns the last event in the schedule
func (es EventSchedule) Last() *ContractEvent {
	if len(es) == 0 {
		return nil
	}
	return &es[len(es)-1]
}

// GetAt returns the event at the specified index
func (es EventSchedule) GetAt(index int) *ContractEvent {
	if index < 0 || index >= len(es) {
		return nil
	}
	return &es[index]
}

// Contains checks if the schedule contains an event of the given type
func (es EventSchedule) Contains(eventType EventType) bool {
	for _, event := range es {
		if event.Type == eventType {
			return true
		}
	}
	return false
}

// NewContractEvent creates a new contract event with the given parameters
func NewContractEvent(eventType EventType, eventTime time.Time, currency string) ContractEvent {
	return ContractEvent{
		Type:         eventType,
		Time:         eventTime,
		ScheduleTime: eventTime, // Default: schedule time = payment time
		Payoff:       decimal.Zero,
		Currency:     currency,
		EventOrder:   EventSequence[eventType],
	}
}
