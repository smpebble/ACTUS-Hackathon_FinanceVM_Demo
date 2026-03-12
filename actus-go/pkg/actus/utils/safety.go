package utils

import (
	"fmt"
	"time"

	"github.com/shopspring/decimal"
)

// Safety limits for contract calculations
const (
	// MaxEventsPerContract is the maximum number of events allowed per contract
	MaxEventsPerContract = 10000

	// MaxDecimalValue is the maximum allowed decimal value to prevent overflow
	MaxDecimalValueStr = "999999999999999999.9999"

	// Date range limits (relative to current time)
	MaxYearsPast   = 100
	MaxYearsFuture = 100
)

var (
	maxDecimalValue = must(decimal.NewFromString(MaxDecimalValueStr))
)

func must(d decimal.Decimal, err error) decimal.Decimal {
	if err != nil {
		panic(fmt.Sprintf("invalid constant: %v", err))
	}
	return d
}

// SafeDiv performs division with zero-check
// Returns error if divisor is zero or result would overflow
func SafeDiv(dividend, divisor decimal.Decimal) (decimal.Decimal, error) {
	if divisor.IsZero() {
		return decimal.Zero, fmt.Errorf("division by zero")
	}

	result := dividend.Div(divisor)

	if result.Abs().GreaterThan(maxDecimalValue) {
		return decimal.Zero, fmt.Errorf("division result overflow")
	}

	return result, nil
}

// SafeMul performs multiplication with overflow check
func SafeMul(a, b decimal.Decimal) (decimal.Decimal, error) {
	result := a.Mul(b)

	if result.Abs().GreaterThan(maxDecimalValue) {
		return decimal.Zero, fmt.Errorf("multiplication overflow")
	}

	return result, nil
}

// SafeAdd performs addition with overflow check
func SafeAdd(a, b decimal.Decimal) (decimal.Decimal, error) {
	result := a.Add(b)

	if result.Abs().GreaterThan(maxDecimalValue) {
		return decimal.Zero, fmt.Errorf("addition overflow")
	}

	return result, nil
}

// SafeSub performs subtraction with overflow check
func SafeSub(a, b decimal.Decimal) (decimal.Decimal, error) {
	result := a.Sub(b)

	if result.Abs().GreaterThan(maxDecimalValue) {
		return decimal.Zero, fmt.Errorf("subtraction overflow")
	}

	return result, nil
}

// ValidateDateRange validates that a date is within reasonable bounds
// Returns error if date is more than MaxYearsPast years in the past
// or more than MaxYearsFuture years in the future
func ValidateDateRange(date time.Time, fieldName string) error {
	now := time.Now()
	minDate := now.AddDate(-MaxYearsPast, 0, 0)
	maxDate := now.AddDate(MaxYearsFuture, 0, 0)

	if date.Before(minDate) {
		return fmt.Errorf("%s is too far in the past (more than %d years): %s",
			fieldName, MaxYearsPast, date.Format("2006-01-02"))
	}

	if date.After(maxDate) {
		return fmt.Errorf("%s is too far in the future (more than %d years): %s",
			fieldName, MaxYearsFuture, date.Format("2006-01-02"))
	}

	return nil
}

// ValidateDateLogic validates logical relationships between dates
func ValidateDateLogic(startDate, endDate time.Time, startName, endName string) error {
	if endDate.Before(startDate) {
		return fmt.Errorf("%s (%s) must be after %s (%s)",
			endName, endDate.Format("2006-01-02"),
			startName, startDate.Format("2006-01-02"))
	}
	return nil
}

// CheckEventLimit checks if the number of events exceeds the safe limit
func CheckEventLimit(currentCount int) error {
	if currentCount >= MaxEventsPerContract {
		return fmt.Errorf("exceeded maximum events limit: %d (max: %d)",
			currentCount, MaxEventsPerContract)
	}
	return nil
}

// SafePow performs power operation with overflow check
// Only supports non-negative integer exponents for safety
func SafePow(base decimal.Decimal, exponent int) (decimal.Decimal, error) {
	if exponent < 0 {
		return decimal.Zero, fmt.Errorf("negative exponents not supported")
	}

	if exponent == 0 {
		return decimal.NewFromInt(1), nil
	}

	result := base
	for i := 1; i < exponent; i++ {
		result = result.Mul(base)
		if result.Abs().GreaterThan(maxDecimalValue) {
			return decimal.Zero, fmt.Errorf("power operation overflow")
		}
	}

	return result, nil
}

// ValidatePositive checks if a decimal value is positive
func ValidatePositive(value decimal.Decimal, fieldName string) error {
	if value.LessThanOrEqual(decimal.Zero) {
		return fmt.Errorf("%s must be positive, got: %s", fieldName, value.String())
	}
	return nil
}

// ValidateNonNegative checks if a decimal value is non-negative
func ValidateNonNegative(value decimal.Decimal, fieldName string) error {
	if value.LessThan(decimal.Zero) {
		return fmt.Errorf("%s must be non-negative, got: %s", fieldName, value.String())
	}
	return nil
}

// SafeCalculate wraps a calculation function with panic recovery
func SafeCalculate(fn func() (decimal.Decimal, error)) (result decimal.Decimal, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("calculation panic recovered: %v", r)
			result = decimal.Zero
		}
	}()
	return fn()
}
