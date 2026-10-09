package money

import (
	"cmp"
	"math/big"
	"slices"
	"sort"
)

// Plus adds amounts in the same currency: price.Plus(shipping, "2.50").
func (m Money) Plus(addends ...any) (Money, error) {
	total := m.big()
	for _, addend := range addends {
		o, err := m.sameCurrencyOperand(addend)
		if err != nil {
			return Money{}, err
		}
		total.Add(total, o.big())
	}

	return m.with(total), nil
}

// Minus subtracts amounts in the same currency: total.Minus(discount).
func (m Money) Minus(subtrahends ...any) (Money, error) {
	total := m.big()
	for _, subtrahend := range subtrahends {
		o, err := m.sameCurrencyOperand(subtrahend)
		if err != nil {
			return Money{}, err
		}
		total.Sub(total, o.big())
	}

	return m.with(total), nil
}

// Times multiplies by an integer or decimal string ("1.5"), rounding to a
// minor unit with the given or configured rounding.
func (m Money) Times(multiplier any, rounding ...Rounding) (Money, error) {
	numerator, denominator, err := numberFraction(multiplier)
	if err != nil {
		return Money{}, err
	}

	return m.divided(new(big.Int).Mul(m.big(), numerator), denominator, rounding)
}

// DividedBy divides by an integer or decimal string ("1.5"), rounding to a
// minor unit. Dividing by zero returns ErrDivisionByZero.
func (m Money) DividedBy(divisor any, rounding ...Rounding) (Money, error) {
	numerator, denominator, err := numberFraction(divisor)
	if err != nil {
		return Money{}, err
	}

	return m.divided(new(big.Int).Mul(m.big(), denominator), numerator, rounding)
}

// Mod returns the remainder after dividing by another amount in the same
// currency; it has the sign of m: Parse("10.00").Mod("3") is 1.00.
func (m Money) Mod(divisor any) (Money, error) {
	o, err := m.sameCurrencyOperand(divisor)
	if err != nil {
		return Money{}, err
	}
	if o.IsZero() {
		return Money{}, divisionByZero()
	}

	return m.with(new(big.Int).Rem(m.big(), o.big())), nil
}

// Negated returns -m.
func (m Money) Negated() Money { return m.with(new(big.Int).Neg(m.big())) }

// Absolute returns |m|.
func (m Money) Absolute() Money {
	if m.IsNegative() {
		return m.Negated()
	}

	return m
}

// MaxScale is the largest scale PercentageOf and RatioOf accept, and the
// largest number of decimals (either sign) RoundTo accepts, so no call can
// force a huge power-of-ten computation. Larger values return
// ErrInvalidOperand. It matches laravel-money's Money::MAX_SCALE.
const MaxScale = 100

// RoundTo rounds to fewer decimals than the currency has and keeps
// minor-unit storage: 12.34 USD RoundTo(0) is 12.00, 15 JPY RoundTo(-1) is
// 20. Decimals at or above the currency's precision return m unchanged.
// Decimals outside -MaxScale..MaxScale, and a rounding that is not a valid
// mode, return ErrInvalidOperand.
func (m Money) RoundTo(decimals int, rounding ...Rounding) (Money, error) {
	if decimals < -MaxScale || decimals > MaxScale {
		return Money{}, newError(ErrInvalidOperand, "the decimals must be between -%d and %d, %d given", MaxScale, MaxScale, decimals)
	}
	if len(rounding) > 0 {
		if err := rounding[0].validate(); err != nil {
			return Money{}, err
		}
	}
	if decimals >= m.currency.minorUnits {
		return m, nil
	}
	mode, err := pick(rounding)
	if err != nil {
		return Money{}, err
	}
	unit := pow10(m.currency.minorUnits - decimals)
	rounded, _ := divide(m.big(), unit, mode) // unit is never zero

	return m.with(rounded.Mul(rounded, unit)), nil
}

// Percent returns percent percent of m: 7.5% of 200.00 is 15.00.
func (m Money) Percent(percent any, rounding ...Rounding) (Money, error) {
	numerator, denominator, err := numberFraction(percent)
	if err != nil {
		return Money{}, err
	}

	return m.divided(new(big.Int).Mul(m.big(), numerator), new(big.Int).Mul(denominator, bigHundred), rounding)
}

// AddPercent returns m plus percent percent of it, e.g. adding 7% tax. The
// percentage is rounded once, before it is added.
func (m Money) AddPercent(percent any, rounding ...Rounding) (Money, error) {
	p, err := m.Percent(percent, rounding...)
	if err != nil {
		return Money{}, err
	}

	return m.Plus(p)
}

// SubtractPercent returns m minus percent percent of it, e.g. a 15% discount.
func (m Money) SubtractPercent(percent any, rounding ...Rounding) (Money, error) {
	p, err := m.Percent(percent, rounding...)
	if err != nil {
		return Money{}, err
	}

	return m.Minus(p)
}

// PercentageOf returns what percentage m is of total as a decimal string with
// scale decimals: 25.00 PercentageOf(200.00, 2) is "12.50". The scale must
// be between 0 and MaxScale.
func (m Money) PercentageOf(total Money, scale int, rounding ...Rounding) (string, error) {
	return m.quotientOf(total, new(big.Int).Mul(m.big(), bigHundred), scale, rounding)
}

// RatioOf returns m divided by other as a decimal string with scale decimals:
// 50.00 RatioOf(200.00, 4) is "0.2500". The scale must be between 0 and
// MaxScale.
func (m Money) RatioOf(other Money, scale int, rounding ...Rounding) (string, error) {
	return m.quotientOf(other, m.big(), scale, rounding)
}

func (m Money) quotientOf(other Money, numerator *big.Int, scale int, rounding []Rounding) (string, error) {
	if _, err := m.sameCurrencyOperand(other); err != nil {
		return "", err
	}
	if other.IsZero() {
		return "", divisionByZero()
	}
	if scale < 0 || scale > MaxScale {
		return "", newError(ErrInvalidOperand, "the scale must be between 0 and %d, %d given", MaxScale, scale)
	}
	mode, err := pick(rounding)
	if err != nil {
		return "", err
	}

	return quotient(numerator, other.big(), scale, mode)
}

func (m Money) divided(numerator, denominator *big.Int, rounding []Rounding) (Money, error) {
	mode, err := pick(rounding)
	if err != nil {
		return Money{}, err
	}
	q, err := divide(numerator, denominator, mode)
	if err != nil {
		return Money{}, err
	}

	return m.with(q), nil
}

// Split divides m into parts equal amounts without losing a minor unit; the
// leftover units go to the first parts: 100.00 split 3 is 33.34, 33.33, 33.33.
func (m Money) Split(parts int) ([]Money, error) {
	if parts < 1 {
		return nil, newError(ErrInvalidAllocation, "money can only be split into one or more parts, %d given", parts)
	}
	ratios := make([]any, parts)
	for i := range ratios {
		ratios[i] = 1
	}

	return m.Allocate(ratios...)
}

// Allocate divides m by ratios (integers or decimal strings such as "0.3")
// without losing a minor unit, returning one amount per ratio in order.
//
// Each part gets its share rounded down; the leftover minor units go, one
// each, to the parts with the largest remainders, earlier parts winning ties,
// exactly like laravel-money and moneyphp. A negative amount is allocated as
// its absolute value and negated. Ratios must be zero or positive and at
// least one must be positive.
func (m Money) Allocate(ratios ...any) ([]Money, error) {
	if len(ratios) == 0 {
		return nil, newError(ErrInvalidAllocation, "cannot allocate money: at least one ratio is required")
	}
	fractions := make([][2]*big.Int, len(ratios))
	for i, ratio := range ratios {
		numerator, denominator, err := numberFraction(ratio)
		if err != nil {
			return nil, err
		}
		fractions[i] = [2]*big.Int{numerator, denominator}
	}

	return m.allocate(fractions)
}

// Ratio is the type of an AllocateMap ratio: an integer or a decimal string.
type Ratio interface {
	~int | ~int8 | ~int16 | ~int32 | ~int64 | ~uint | ~uint8 | ~uint16 | ~uint32 | ~uint64 | ~string
}

// AllocateMap is Allocate with named ratios, returning one amount per key:
//
//	shares, err := money.AllocateMap(total, map[string]int{"owner": 70, "agent": 20, "platform": 10})
//
// Go maps have no order, so leftover minor units that tie go to keys in
// ascending key order, which keeps the result deterministic.
func AllocateMap[K cmp.Ordered, R Ratio](m Money, ratios map[K]R) (map[K]Money, error) {
	keys := make([]K, 0, len(ratios))
	for key := range ratios {
		keys = append(keys, key)
	}
	slices.Sort(keys)
	ordered := make([]any, len(keys))
	for i, key := range keys {
		ordered[i] = ratios[key]
	}
	shares, err := m.Allocate(ordered...)
	if err != nil {
		return nil, err
	}
	out := make(map[K]Money, len(keys))
	for i, key := range keys {
		out[key] = shares[i]
	}

	return out, nil
}

func (m Money) allocate(fractions [][2]*big.Int) ([]Money, error) {
	scale := 0
	for _, f := range fractions {
		if digits := len(f[1].String()) - 1; digits > scale {
			scale = digits
		}
	}
	unit := pow10(scale)
	weights := make([]*big.Int, len(fractions))
	total := new(big.Int)
	for i, f := range fractions {
		if f[0].Sign() < 0 {
			return nil, newError(ErrInvalidAllocation, "cannot allocate money: ratios must be zero or positive")
		}
		weights[i] = new(big.Int).Mul(f[0], new(big.Int).Quo(unit, f[1]))
		total.Add(total, weights[i])
	}
	if total.Sign() == 0 {
		return nil, newError(ErrInvalidAllocation, "cannot allocate money: the sum of the ratios must be greater than zero")
	}

	absolute := new(big.Int).Abs(m.big())
	shares := make([]*big.Int, len(weights))
	remainders := make([]*big.Int, len(weights))
	left := new(big.Int).Set(absolute)
	for i, weight := range weights {
		product := new(big.Int).Mul(absolute, weight)
		shares[i], remainders[i] = new(big.Int).QuoRem(product, total, new(big.Int))
		left.Sub(left, shares[i])
	}

	order := make([]int, len(weights))
	for i := range order {
		order[i] = i
	}
	sort.SliceStable(order, func(a, b int) bool { return remainders[order[a]].Cmp(remainders[order[b]]) > 0 })
	// left < len(weights): each share lost less than one unit.
	for _, i := range order[:left.Int64()] {
		shares[i].Add(shares[i], bigOne)
	}

	negative := m.IsNegative()
	out := make([]Money, len(shares))
	for i, share := range shares {
		if negative {
			share.Neg(share)
		}
		out[i] = m.with(share)
	}

	return out, nil
}

// Sum returns the total of one or more amounts in the same currency.
func Sum(monies []Money) (Money, error) {
	if len(monies) == 0 {
		return Money{}, emptyAggregate("Sum")
	}
	rest := make([]any, len(monies)-1)
	for i, m := range monies[1:] {
		rest[i] = m
	}

	return monies[0].Plus(rest...)
}

// Min returns the smallest of one or more amounts in the same currency.
func Min(monies []Money) (Money, error) {
	return extreme(monies, "Min", -1)
}

// Max returns the largest of one or more amounts in the same currency.
func Max(monies []Money) (Money, error) {
	return extreme(monies, "Max", 1)
}

// Avg returns the average of one or more amounts in the same currency,
// rounded to a minor unit with the given or configured rounding.
func Avg(monies []Money, rounding ...Rounding) (Money, error) {
	total, err := Sum(monies)
	if err != nil {
		if len(monies) == 0 {
			return Money{}, emptyAggregate("Avg")
		}

		return Money{}, err
	}

	return total.DividedBy(len(monies), rounding...)
}

func extreme(monies []Money, operation string, want int) (Money, error) {
	if len(monies) == 0 {
		return Money{}, emptyAggregate(operation)
	}
	result := monies[0]
	for _, m := range monies[1:] {
		c, err := m.Compare(result)
		if err != nil {
			return Money{}, err
		}
		if c == want {
			result = m
		}
	}

	return result, nil
}

func emptyAggregate(operation string) error {
	return newError(ErrEmptyAggregate, "money.%s needs at least one Money", operation)
}
