package utils

import (
	"time"

	"github.com/shopspring/decimal"
	"github.com/yourusername/actus-go/pkg/actus/types"
)

// Package-level cached decimal constants for performance optimization
// These are used frequently in day count calculations to avoid repeated allocations
var (
	decimal365 = decimal.NewFromInt(365)
	decimal360 = decimal.NewFromInt(360)
)

// YearFraction calculates the year fraction between two dates
// according to the specified day count convention
// This implements ACTUS specification section 3.3
func YearFraction(start, end time.Time, convention types.DayCountConvention) decimal.Decimal {
	// Handle edge cases
	if end.Before(start) || end.Equal(start) {
		return decimal.Zero
	}

	// Normalize convention string to handle variations
	// ACTUS tests use formats without slashes (e.g., "30E360" instead of "30E/360")
	normalizedConvention := normalizeConvention(convention)

	switch normalizedConvention {
	case types.DCC_A_360:
		return actualOver360(start, end)
	case types.DCC_A_365:
		return actualOver365(start, end)
	case types.DCC_30E360:
		return thirtyEOver360(start, end)
	case types.DCC_30_360:
		return thirtyOver360(start, end)
	case types.DCC_A_A:
		return actualOverActual(start, end)
	default:
		// Default to Actual/365
		return actualOver365(start, end)
	}
}

// normalizeConvention converts various day count convention formats to standard format
func normalizeConvention(convention types.DayCountConvention) types.DayCountConvention {
	convStr := string(convention)

	// Handle common variations
	switch convStr {
	case "30E360":
		return types.DCC_30E360 // "30E/360"
	case "A360":
		return types.DCC_A_360 // "A/360"
	case "A365":
		return types.DCC_A_365 // "A/365"
	case "AA", "AAA":
		return types.DCC_A_A // "A/A"
	default:
		return convention
	}
}

// actualOver360 implements Actual/360 day count convention
// Formula: (actual days between dates) / 360
func actualOver360(start, end time.Time) decimal.Decimal {
	days := calculateActualDays(start, end)
	return decimal.NewFromInt(int64(days)).Div(decimal360)
}

// actualOver365 implements Actual/365 day count convention
// Formula: (actual days between dates) / 365
func actualOver365(start, end time.Time) decimal.Decimal {
	days := calculateActualDays(start, end)
	return decimal.NewFromInt(int64(days)).Div(decimal365)
}

// calculateActualDays returns the number of actual days between two dates
// Handles edge cases like 23:59:59 timestamps which should count as a full day
func calculateActualDays(start, end time.Time) int {
	hours := end.Sub(start).Hours()
	daysFloat := hours / 24
	days := int(daysFloat)

	// If we're very close to the next integer (within 1 minute of a full day),
	// round up. This handles timestamps like 23:59:59 which should count as a full day.
	fractional := daysFloat - float64(days)
	if fractional > 0.999 { // More than 23:58:33 into the day
		days++
	}
	return days
}

// thirtyEOver360 implements 30E/360 (Eurobond basis) day count convention
// Formula: (360*(Y2-Y1) + 30*(M2-M1) + (D2-D1)) / 360
// where days are adjusted: if D1 or D2 is 31, treat as 30
func thirtyEOver360(start, end time.Time) decimal.Decimal {
	y1, m1, d1 := start.Date()
	y2, m2, d2 := end.Date()

	// Adjust day values according to 30E/360 rules
	if d1 == 31 {
		d1 = 30
	}
	if d2 == 31 {
		d2 = 30
	}

	// Calculate total days using 30-day months
	days := 360*(y2-y1) + 30*(int(m2)-int(m1)) + (d2 - d1)

	return decimal.NewFromInt(int64(days)).Div(decimal.NewFromInt(360))
}

// thirtyOver360 implements 30/360 (US) day count convention
// Formula: (360*(Y2-Y1) + 30*(M2-M1) + (D2-D1)) / 360
// with specific adjustment rules for month-end dates
func thirtyOver360(start, end time.Time) decimal.Decimal {
	y1, m1, d1 := start.Date()
	y2, m2, d2 := end.Date()

	// US 30/360 adjustment rules
	if d1 == 31 {
		d1 = 30
	}
	if d2 == 31 && d1 >= 30 {
		d2 = 30
	}

	// Calculate total days using 30-day months
	days := 360*(y2-y1) + 30*(int(m2)-int(m1)) + (d2 - d1)

	return decimal.NewFromInt(int64(days)).Div(decimal.NewFromInt(360))
}

// actualOverActual implements Actual/Actual (ISDA) day count convention
// This is the most complex convention as it accounts for leap years
// ISDA definition: sum of (days in each year portion / days in that year)
func actualOverActual(start, end time.Time) decimal.Decimal {
	// If within same year, simple calculation
	if start.Year() == end.Year() {
		daysInYear := daysInYear(start.Year())
		actualDays := end.Sub(start).Hours() / 24
		return decimal.NewFromFloat(actualDays).Div(decimal.NewFromInt(int64(daysInYear)))
	}

	// For dates spanning multiple years, calculate per year using precise integer arithmetic
	// to avoid floating-point precision issues

	// Calculate days in the first partial year (from start to end of year)
	// Use start of next year (midnight Jan 1) for precise day counting
	startOfNextYear := time.Date(start.Year()+1, 1, 1, 0, 0, 0, 0, start.Location())
	daysInFirstYear := daysInYear(start.Year())
	daysInFirstPeriod := int(startOfNextYear.Sub(start).Hours() / 24)

	// Calculate days in the last partial year (from start of year to end)
	startOfLastYear := time.Date(end.Year(), 1, 1, 0, 0, 0, 0, end.Location())
	daysInLastYear := daysInYear(end.Year())
	daysInLastPeriod := int(end.Sub(startOfLastYear).Hours() / 24)

	// Use decimal arithmetic for precision
	// Year fraction = (daysInFirstPeriod / daysInFirstYear) + complete years + (daysInLastPeriod / daysInLastYear)
	yearFrac := decimal.NewFromInt(int64(daysInFirstPeriod)).Div(decimal.NewFromInt(int64(daysInFirstYear)))

	// Add complete years in between
	// Optimized: Calculate complete years directly instead of looping
	// For multi-year periods, this reduces allocations from O(n) to O(1)
	completeYears := end.Year() - start.Year() - 1
	if completeYears > 0 {
		yearFrac = yearFrac.Add(decimal.NewFromInt(int64(completeYears)))
	}

	// Add fraction for the last partial year
	yearFrac = yearFrac.Add(
		decimal.NewFromInt(int64(daysInLastPeriod)).Div(decimal.NewFromInt(int64(daysInLastYear))),
	)

	return yearFrac
}

// daysInYear returns the number of days in the given year (365 or 366)
func daysInYear(year int) int {
	if isLeapYear(year) {
		return 366
	}
	return 365
}

// isLeapYear determines if a year is a leap year
// Leap year rules:
// - Divisible by 4: leap year
// - Divisible by 100: not a leap year
// - Divisible by 400: leap year
func isLeapYear(year int) bool {
	return year%4 == 0 && (year%100 != 0 || year%400 == 0)
}
