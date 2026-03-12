package utils

import (
	"fmt"
	"regexp"
	"strconv"
	"sync"
	"time"
)

// cycleCache stores parsed cycles to avoid repeated regex parsing
// Key: cycle string (e.g., "P3M"), Value: *Cycle
var cycleCache sync.Map

// Cycle represents a periodic interval according to ISO 8601
type Cycle struct {
	N      int    // Number of periods
	Period string // Period unit: D, W, M, Q, H, Y
	Stub   string // Stub type: "+" (long last), "-" (short last), "" (none)
}

// Cycle pattern regex: P<n><period>[<stub>]
// Examples: P3M, P1Y, P6M+, P1M-, P1ML0, P3ML1
// ACTUS format also supports: L<n> where L=Last day of month, n=stub length
var cycleRegex = regexp.MustCompile(`^P(\d+)([DWMQHY])(.*)$`)

// ParseCycle parses a cycle string into a Cycle struct
// Format: P<n><period>[<stub>]
// Examples:
//   - "P3M"   = Every 3 months
//   - "P1Y"   = Every year
//   - "P6M+"  = Every 6 months with long last stub
//   - "P1M-"  = Every month with short last stub
//   - "P1ML0" = Every month, last day of month, no stub (ACTUS format)
//   - "P3ML1" = Every 3 months, last day of month, stub length 1
func ParseCycle(cycleStr string) (*Cycle, error) {
	if cycleStr == "" {
		return nil, nil
	}

	// Check cache first to avoid regex parsing overhead
	if cached, ok := cycleCache.Load(cycleStr); ok {
		return cached.(*Cycle), nil
	}

	matches := cycleRegex.FindStringSubmatch(cycleStr)
	if len(matches) < 3 {
		return nil, fmt.Errorf("invalid cycle format: %s (expected format: P<n><period>, e.g., P3M)", cycleStr)
	}

	n, err := strconv.Atoi(matches[1])
	if err != nil || n <= 0 {
		return nil, fmt.Errorf("invalid cycle number: %s (must be positive integer)", matches[1])
	}

	period := matches[2]
	stub := ""
	if len(matches) > 3 && matches[3] != "" {
		stubStr := matches[3]
		// Handle ACTUS format: L<n> where L=Last day of month
		// L0 = Remove short last stub (like "-")
		// L1 = Long last stub (like "+")
		if stubStr == "+" || stubStr == "-" {
			stub = stubStr
		} else if len(stubStr) > 0 && stubStr[0] == 'L' {
			// ACTUS format like "L0", "L1"
			if stubStr == "L0" {
				// L0 = Remove short last stub
				stub = "-"
			} else if stubStr == "L1" {
				// L1 = Long last stub
				stub = "+"
			} else {
				// L2 or other = No stub modification
				stub = ""
			}
		} else {
			stub = stubStr
		}
	}

	cycle := &Cycle{
		N:      n,
		Period: period,
		Stub:   stub,
	}

	// Store in cache for future lookups
	cycleCache.Store(cycleStr, cycle)

	return cycle, nil
}

// AddCycle adds the cycle interval to the given date
// For month-based periods (M, Q, H, Y), it preserves the day of month
// and uses the last day of month if the target month doesn't have enough days
func AddCycle(date time.Time, cycle *Cycle) time.Time {
	if cycle == nil {
		return date
	}

	switch cycle.Period {
	case "D": // Day
		return date.AddDate(0, 0, cycle.N)
	case "W": // Week
		return date.AddDate(0, 0, cycle.N*7)
	case "M": // Month
		return addMonthsPreservingDay(date, cycle.N)
	case "Q": // Quarter
		return addMonthsPreservingDay(date, cycle.N*3)
	case "H": // Half-year
		return addMonthsPreservingDay(date, cycle.N*6)
	case "Y": // Year
		return addMonthsPreservingDay(date, cycle.N*12)
	default:
		return date
	}
}

// addMonthsPreservingDay adds months to a date while preserving the day of month
// If the target month doesn't have enough days, use the last day of that month
func addMonthsPreservingDay(date time.Time, months int) time.Time {
	// Save the original day
	originalDay := date.Day()

	// Calculate the target year and month
	targetYear := date.Year()
	targetMonth := int(date.Month()) + months

	// Handle year overflow
	for targetMonth > 12 {
		targetYear++
		targetMonth -= 12
	}
	for targetMonth < 1 {
		targetYear--
		targetMonth += 12
	}

	// Get the last day of the target month
	lastDay := time.Date(targetYear, time.Month(targetMonth)+1, 0, 0, 0, 0, 0, time.UTC).Day()

	// Use original day if possible, otherwise use last day of month
	actualDay := originalDay
	if originalDay > lastDay {
		actualDay = lastDay
	}

	return time.Date(
		targetYear,
		time.Month(targetMonth),
		actualDay,
		date.Hour(),
		date.Minute(),
		date.Second(),
		date.Nanosecond(),
		date.Location(),
	)
}

// GenerateCyclicDates generates a sequence of dates based on the cycle
// starting from anchor until end (inclusive based on includeEnd parameter)
func GenerateCyclicDates(anchor time.Time, cycle *Cycle, end time.Time, includeEnd bool) []time.Time {
	return GenerateCyclicDatesWithEOM(anchor, cycle, end, includeEnd, false)
}

// GenerateCyclicDatesWithEOM generates a sequence of dates based on the cycle
// with optional End of Month convention support
func GenerateCyclicDatesWithEOM(anchor time.Time, cycle *Cycle, end time.Time, includeEnd bool, eom bool) []time.Time {
	if cycle == nil || anchor.After(end) {
		return []time.Time{}
	}

	dates := []time.Time{}
	current := anchor

	// Check if anchor is at end of month for EOM handling
	isAnchorEOM := eom && isEndOfMonth(anchor)

	// Generate dates until we reach or exceed the end date
	for {
		// Always include the first date (anchor)
		dates = append(dates, current)

		// Calculate next date
		next := AddCycle(current, cycle)

		// Apply EOM convention if needed
		if isAnchorEOM {
			next = adjustToEndOfMonth(next)
		}

		// Stop if next date exceeds end
		if next.After(end) {
			break
		}

		current = next
	}

	// Handle stub conventions and end date
	lastGenerated := dates[len(dates)-1]

	if !lastGenerated.Equal(end) {
		switch cycle.Stub {
		case "-": // Short last stub: remove last period if it doesn't match end
			if len(dates) > 1 {
				dates = dates[:len(dates)-1]
			}
		case "+": // Long last stub: include end date
			if !lastGenerated.Equal(end) {
				dates = append(dates, end)
			}
		default:
			// No stub specified
			if includeEnd && !lastGenerated.Equal(end) {
				dates = append(dates, end)
			}
		}
	}

	return dates
}

// isEndOfMonth checks if the given date is the last day of its month
func isEndOfMonth(date time.Time) bool {
	// Get the last day of the current month
	year, month, _ := date.Date()
	lastDay := time.Date(year, month+1, 0, 0, 0, 0, 0, time.UTC).Day()
	return date.Day() == lastDay
}

// adjustToEndOfMonth adjusts a date to the end of its month
func adjustToEndOfMonth(date time.Time) time.Time {
	year, month, _ := date.Date()
	lastDay := time.Date(year, month+1, 0, 0, 0, 0, 0, time.UTC).Day()
	return time.Date(year, month, lastDay, date.Hour(), date.Minute(), date.Second(), date.Nanosecond(), date.Location())
}

// CountPeriods counts the number of periods between start and end dates
func CountPeriods(start, end time.Time, cycle *Cycle) int {
	if cycle == nil || start.After(end) {
		return 0
	}

	count := 0
	current := start

	for current.Before(end) {
		current = AddCycle(current, cycle)
		count++
	}

	return count
}

// String returns a string representation of the cycle
func (c *Cycle) String() string {
	if c == nil {
		return ""
	}
	return fmt.Sprintf("P%d%s%s", c.N, c.Period, c.Stub)
}
