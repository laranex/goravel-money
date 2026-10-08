package money

import (
	"math"
	"strconv"
	"strings"
)

// Parse converts a plain decimal string such as "10.50", "-0.5", ".5" or "2500"
// into Money using the currency's minor units. It follows moneyphp's
// DecimalMoneyParser (used by laravel-money): surrounding whitespace is
// trimmed, an empty string is zero, and extra fraction digits are rounded half
// away from zero ("10.555" USD is 1056). Signs other than a leading "-",
// thousands separators and exponents are rejected with ErrInvalidDecimal.
func Parse(decimal string, currency Currency) (Money, error) {
	amount, err := parseMinor(decimal, currency.minorUnits)
	if err != nil {
		return Money{}, err
	}

	return New(amount, currency), nil
}

func parseMinor(input string, minorUnits int) (int64, error) {
	s := strings.TrimSpace(input)
	if s == "" {
		return 0, nil
	}

	negative := false
	if s[0] == '-' {
		negative = true
		s = s[1:]
	}
	whole, fraction, _ := strings.Cut(s, ".")
	if whole+fraction == "" || !allDigits(whole) || !allDigits(fraction) {
		return 0, newError(ErrInvalidDecimal, "cannot parse %q as a decimal amount", input)
	}

	roundUp := false
	if len(fraction) > minorUnits {
		roundUp = fraction[minorUnits] >= '5'
		fraction = fraction[:minorUnits]
	} else {
		fraction += zeros(minorUnits - len(fraction))
	}

	digits := strings.TrimLeft(whole+fraction, "0")
	var magnitude uint64
	if digits != "" {
		var err error
		magnitude, err = strconv.ParseUint(digits, 10, 64)
		if err != nil {
			return 0, newError(ErrOverflow, "%q does not fit in int64 minor units", input)
		}
	}
	if roundUp {
		if magnitude == math.MaxUint64 {
			return 0, newError(ErrOverflow, "%q does not fit in int64 minor units", input)
		}
		magnitude++
	}

	if negative {
		if magnitude > uint64(math.MaxInt64)+1 {
			return 0, newError(ErrOverflow, "%q does not fit in int64 minor units", input)
		}

		return int64(-magnitude), nil
	}
	if magnitude > math.MaxInt64 {
		return 0, newError(ErrOverflow, "%q does not fit in int64 minor units", input)
	}

	return int64(magnitude), nil
}

func allDigits(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] < '0' || s[i] > '9' {
			return false
		}
	}

	return true
}
