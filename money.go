package money

import (
	"math/big"
	"strings"
)

// Money is an immutable amount of one currency, held as an exact integer
// number of minor units (cents) of any size. Amounts never pass through a
// float. Every method returns a new value.
//
// Money is comparable, so == works and Money can be a map key, but prefer
// Equals, which also accepts decimal strings.
//
// Wherever a method takes an operand of type any, the operand is a Money, or a
// decimal amount in this money's currency given as a string ("2.50"), an
// integer (2 means 2.00) or a *big.Int. Floats are rejected with
// ErrInvalidOperand. Multipliers, divisors, percentages and ratios are decimal
// strings or integers.
type Money struct {
	amount   string // canonical integer minor units: no leading zeros, "0", "-5"
	currency Currency
}

// New returns Money for an amount in minor units: New(1050, usd) is 10.50 USD.
func New(minor int64, currency Currency) Money {
	return FromBigInt(big.NewInt(minor), currency)
}

// FromBigInt returns Money for an amount in minor units of any size.
func FromBigInt(minor *big.Int, currency Currency) Money {
	if minor == nil {
		return Money{amount: "0", currency: currency}
	}

	return Money{amount: minor.String(), currency: currency}
}

// OfMinor returns Money for an amount in minor units given as an integer
// string of any size: OfMinor("123450", usd) is 1,234.50 USD. Anything but an
// optional sign and digits returns ErrInvalidOperand.
func OfMinor(minor string, currency Currency) (Money, error) {
	trimmed := strings.TrimSpace(minor)
	digits := strings.TrimLeft(trimmed, "+-")
	if len(trimmed)-len(digits) > 1 || !isDigits(digits) {
		return Money{}, newError(ErrInvalidOperand, "minor-unit amounts must be integers such as 1050 or \"1050\", %q given; use money.Parse for decimal amounts", minor)
	}

	return FromBigInt(mustBig(trimmed), currency), nil
}

// Parse returns Money for a decimal amount such as "1234.50", "1,234.50",
// "-0.5" or "12" in the given currency.
//
// Parsing is strict, like laravel-money: only digits, an optional sign, one
// dot as the decimal separator, and commas or spaces grouping thousands are
// accepted ("12,50" is an error, not 1250). More decimals than the currency
// has return a *ParseError (ErrTooManyDecimals) unless a rounding mode is
// given; extra decimals that are all zeros are always accepted.
func Parse(amount string, currency Currency, rounding ...Rounding) (Money, error) {
	var mode *Rounding
	if len(rounding) > 0 {
		mode = &rounding[0]
	}
	minor, err := toMinor(amount, currency.minorUnits, mode, currency.code)
	if err != nil {
		return Money{}, err
	}

	return FromBigInt(minor, currency), nil
}

// MustParse is like Parse but panics on an error. Use it for constants.
func MustParse(amount string, currency Currency, rounding ...Rounding) Money {
	m, err := Parse(amount, currency, rounding...)
	if err != nil {
		panic(err)
	}

	return m
}

// Zero returns zero in the given currency.
func Zero(currency Currency) Money {
	return Money{amount: "0", currency: currency}
}

// Amount returns the amount in minor units as an integer string, for example
// "123450" for 1,234.50 USD.
func (m Money) Amount() string {
	if m.amount == "" {
		return "0"
	}

	return m.amount
}

// Int64 returns the amount in minor units, or ErrOverflow when it does not fit
// in an int64.
func (m Money) Int64() (int64, error) {
	n := m.big()
	if !n.IsInt64() {
		return 0, newError(ErrOverflow, "%s does not fit in int64 minor units", m)
	}

	return n.Int64(), nil
}

// BigInt returns the amount in minor units as a new *big.Int.
func (m Money) BigInt() *big.Int { return m.big() }

// Decimal returns the amount as a decimal string with the currency's
// precision: "1234.50" for USD, "2500" for JPY, "-0.005" for KWD.
func (m Money) Decimal() string { return fromMinor(m.big(), m.currency.minorUnits) }

// Currency returns the currency.
func (m Money) Currency() Currency { return m.currency }

// Precision returns the number of decimal places of the currency.
func (m Money) Precision() int { return m.currency.minorUnits }

// Sign returns -1, 0 or +1.
func (m Money) Sign() int {
	switch {
	case m.amount == "" || m.amount == "0":
		return 0
	case m.amount[0] == '-':
		return -1
	default:
		return 1
	}
}

// IsZero reports whether the amount is zero.
func (m Money) IsZero() bool { return m.Sign() == 0 }

// IsPositive reports whether the amount is greater than zero.
func (m Money) IsPositive() bool { return m.Sign() > 0 }

// IsNegative reports whether the amount is less than zero.
func (m Money) IsNegative() bool { return m.Sign() < 0 }

// IsSameCurrency reports whether every given amount has m's currency.
func (m Money) IsSameCurrency(others ...Money) bool {
	for _, other := range others {
		if !m.currency.Equals(other.currency) {
			return false
		}
	}

	return true
}

// Equals reports whether other has the same currency and amount. Different
// currencies, and operands that are not valid amounts, are never equal.
func (m Money) Equals(other any) bool {
	o, err := m.operand(other)
	if err != nil {
		return false
	}

	return m.IsSameCurrency(o) && m.big().Cmp(o.big()) == 0
}

// Compare returns -1, 0 or +1 as m is less than, equal to or greater than
// other. A different currency returns a *CurrencyMismatchError.
func (m Money) Compare(other any) (int, error) {
	o, err := m.sameCurrencyOperand(other)
	if err != nil {
		return 0, err
	}

	return m.big().Cmp(o.big()), nil
}

// GreaterThan reports whether m > other.
func (m Money) GreaterThan(other any) (bool, error) {
	c, err := m.Compare(other)

	return c > 0, err
}

// GreaterThanOrEqual reports whether m >= other.
func (m Money) GreaterThanOrEqual(other any) (bool, error) {
	c, err := m.Compare(other)

	return err == nil && c >= 0, err
}

// LessThan reports whether m < other.
func (m Money) LessThan(other any) (bool, error) {
	c, err := m.Compare(other)

	return c < 0, err
}

// LessThanOrEqual reports whether m <= other.
func (m Money) LessThanOrEqual(other any) (bool, error) {
	c, err := m.Compare(other)

	return err == nil && c <= 0, err
}

// String returns the currency code and the decimal amount, "USD 1234.50".
// It does not depend on config; use Format for locale-aware output.
func (m Money) String() string {
	return m.currency.code + " " + m.Decimal()
}

// Format returns the amount formatted for display, for example "$1,234.50",
// in the given locale or, without one, the configured locale (money.locale,
// then app.locale). Without a supported locale it returns "USD 1234.50".
func (m Money) Format(locale ...string) string {
	manager, err := current()
	if err != nil {
		return m.String()
	}

	return manager.Format(m, locale...)
}

func (m Money) big() *big.Int {
	if m.amount == "" {
		return new(big.Int)
	}

	return mustBig(m.amount)
}

func (m Money) with(minor *big.Int) Money {
	return Money{amount: minor.String(), currency: m.currency}
}

// operand turns a Money or a decimal amount in m's currency into Money.
func (m Money) operand(value any) (Money, error) {
	if money, ok := value.(Money); ok {
		return money, nil
	}
	if money, ok := value.(*Money); ok && money != nil {
		return *money, nil
	}
	text, err := numberString(value)
	if err != nil {
		return Money{}, err
	}

	return Parse(text, m.currency)
}

func (m Money) sameCurrencyOperand(value any) (Money, error) {
	o, err := m.operand(value)
	if err != nil {
		return Money{}, err
	}
	if !m.IsSameCurrency(o) {
		left := m

		return Money{}, &CurrencyMismatchError{Left: &left, Expected: m.currency.code, Given: o}
	}

	return o, nil
}
