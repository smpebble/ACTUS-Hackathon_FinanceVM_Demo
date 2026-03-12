package utils

import (
	"github.com/shopspring/decimal"
)

// ===== Cached Decimal Constants =====
// These constants are pre-allocated to avoid repeated memory allocation
// during contract calculations. Used across all 18 contract types.

var (
	// DecimalZero is the cached zero value
	DecimalZero = decimal.Zero

	// DecimalOne is the cached value of 1
	DecimalOne = decimal.NewFromInt(1)

	// DecimalNegOne is the cached value of -1
	DecimalNegOne = decimal.NewFromInt(-1)

	// DecimalTwo is the cached value of 2
	DecimalTwo = decimal.NewFromInt(2)

	// DecimalHundred is the cached value of 100
	DecimalHundred = decimal.NewFromInt(100)

	// Decimal360 is the cached value of 360 (for day count conventions)
	Decimal360 = decimal.NewFromInt(360)

	// Decimal365 is the cached value of 365 (for day count conventions)
	Decimal365 = decimal.NewFromInt(365)
)

// RoleSign returns the cached decimal for contract role sign
// +1 for asset positions (RPA, LG, BUY, RFL)
// -1 for liability positions (RPL, ST, SEL, PFL)
func RoleSign(role string) decimal.Decimal {
	switch role {
	case "RPA", "LG", "BUY", "RFL":
		return DecimalOne
	case "RPL", "ST", "SEL", "PFL":
		return DecimalNegOne
	default:
		return DecimalOne
	}
}
