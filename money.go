package money

import (
	"bytes"
	"encoding/json"
	"math"
	"strconv"
)

// Money is an amount in integer minor units (cents) of one ISO 4217 currency.
// It is an immutable value type; amounts never pass through a float.
type Money struct {
	amount   int64
	currency Currency
}

// New returns Money for an amount in minor units, so New(1050, usd) is 10.50 USD.
func New(amount int64, currency Currency) Money {
	return Money{amount: amount, currency: currency}
}

// Amount returns the amount in minor units.
func (m Money) Amount() int64 { return m.amount }

// Currency returns the currency.
func (m Money) Currency() Currency { return m.currency }

// IsZero reports whether the amount is zero.
func (m Money) IsZero() bool { return m.amount == 0 }

// IsPositive reports whether the amount is greater than zero.
func (m Money) IsPositive() bool { return m.amount > 0 }

// IsNegative reports whether the amount is less than zero.
func (m Money) IsNegative() bool { return m.amount < 0 }

// SameCurrency reports whether m and other share a currency.
func (m Money) SameCurrency(other Money) bool { return m.currency.Equals(other.currency) }

// Equals reports whether m and other have the same amount and currency.
func (m Money) Equals(other Money) bool {
	return m.amount == other.amount && m.SameCurrency(other)
}

// Compare returns -1, 0 or +1 as m is less than, equal to or greater than
// other. Different currencies return an error wrapping ErrCurrencyMismatch.
func (m Money) Compare(other Money) (int, error) {
	if err := m.assertSameCurrency(other); err != nil {
		return 0, err
	}

	switch {
	case m.amount < other.amount:
		return -1, nil
	case m.amount > other.amount:
		return 1, nil
	default:
		return 0, nil
	}
}

// Add returns m + other. Different currencies return ErrCurrencyMismatch and
// results outside int64 return ErrOverflow.
func (m Money) Add(other Money) (Money, error) {
	if err := m.assertSameCurrency(other); err != nil {
		return Money{}, err
	}
	sum := m.amount + other.amount
	if (other.amount > 0 && sum < m.amount) || (other.amount < 0 && sum > m.amount) {
		return Money{}, newError(ErrOverflow, "%s + %s overflows int64 minor units", m, other)
	}

	return New(sum, m.currency), nil
}

// Subtract returns m - other with the same rules as Add.
func (m Money) Subtract(other Money) (Money, error) {
	if err := m.assertSameCurrency(other); err != nil {
		return Money{}, err
	}
	diff := m.amount - other.amount
	if (other.amount < 0 && diff < m.amount) || (other.amount > 0 && diff > m.amount) {
		return Money{}, newError(ErrOverflow, "%s - %s overflows int64 minor units", m, other)
	}

	return New(diff, m.currency), nil
}

// Decimal returns the amount as a plain decimal string using the currency's
// minor units: "10.50" for 1050 USD, "2500" for 2500 JPY, "-0.05" for -5 USD.
func (m Money) Decimal() string {
	return formatDecimal(m.amount, m.currency.minorUnits)
}

// String returns the decimal amount followed by the currency code, "10.50 USD".
func (m Money) String() string {
	return m.Decimal() + " " + m.currency.code
}

type moneyJSON struct {
	Amount   json.RawMessage `json:"amount"`
	Currency string          `json:"currency"`
}

// MarshalJSON encodes {"amount":"1050","currency":"USD"}: minor units as a
// string (the same shape as moneyphp/laravel-money), safe for JavaScript.
func (m Money) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		Amount   string `json:"amount"`
		Currency string `json:"currency"`
	}{strconv.FormatInt(m.amount, 10), m.currency.code})
}

// UnmarshalJSON decodes {"amount":..., "currency":...}; the amount may be an
// integer string or an integer number of minor units.
func (m *Money) UnmarshalJSON(data []byte) error {
	var raw moneyJSON
	if err := json.Unmarshal(data, &raw); err != nil {
		return newError(ErrInvalidStoredAmount, "cannot decode money JSON: %v", err)
	}
	currency, err := LookupCurrency(raw.Currency)
	if err != nil {
		return err
	}
	amount, err := amountFromJSON(raw.Amount)
	if err != nil {
		return err
	}
	*m = New(amount, currency)

	return nil
}

func amountFromJSON(raw json.RawMessage) (int64, error) {
	raw = bytes.TrimSpace(raw)
	if len(raw) > 0 && raw[0] == '"' {
		var s string
		if err := json.Unmarshal(raw, &s); err != nil {
			return 0, newError(ErrInvalidStoredAmount, "invalid amount %s", raw)
		}
		raw = []byte(s)
	}
	amount, err := strconv.ParseInt(string(raw), 10, 64)
	if err != nil {
		return 0, newError(ErrInvalidStoredAmount, "amount must be an integer in minor units, got %q", string(raw))
	}

	return amount, nil
}

func (m Money) assertSameCurrency(other Money) error {
	if !m.SameCurrency(other) {
		return newError(ErrCurrencyMismatch, "cannot combine %s with %s", m.currency.code, other.currency.code)
	}

	return nil
}

func formatDecimal(amount int64, minorUnits int) string {
	negative := amount < 0
	var digits string
	if amount == math.MinInt64 {
		digits = "9223372036854775808"
	} else {
		if negative {
			amount = -amount
		}
		digits = strconv.FormatInt(amount, 10)
	}

	var out string
	switch {
	case minorUnits == 0:
		out = digits
	case len(digits) > minorUnits:
		out = digits[:len(digits)-minorUnits] + "." + digits[len(digits)-minorUnits:]
	default:
		out = "0." + zeros(minorUnits-len(digits)) + digits
	}
	if negative {
		out = "-" + out
	}

	return out
}

func zeros(n int) string {
	b := make([]byte, n)
	for i := range b {
		b[i] = '0'
	}

	return string(b)
}
