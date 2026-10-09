package money

import (
	"math/big"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// The tests in this file mirror laravel-money's tests/Feature/ParityTest.php:
// the same input gives the same result, and the same kind of error, in both
// packages.

func TestParityOf(t *testing.T) {
	useConfig(t, laravelConfig())
	usd := MustCurrency("USD")
	manager, err := current()
	require.NoError(t, err)

	tests := []struct {
		name   string
		amount any
		code   string
		want   string
	}{
		{"decimal string", "1,234.50", "USD", "USD 1234.50"},
		{"integer is a whole amount", 1234, "JPY", "JPY 1234"},
		{"negative integer", -5, "USD", "USD -5.00"},
		{"big.Int is a whole amount", big.NewInt(7), "KWD", "KWD 7.000"},
		{"default currency", "12.34", "", "USD 12.34"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m, err := manager.Of(tt.amount, tt.code)
			require.NoError(t, err)
			assert.Equal(t, tt.want, m.String())
		})
	}

	assert.Equal(t, "1.24", MustOf("1.235", usd, HalfUp).Decimal())
	assert.Panics(t, func() { MustOf("1.235", usd) })

	_, err = Of(12.5, usd)
	assert.ErrorIs(t, err, ErrInvalidOperand)
	assert.ErrorContains(t, err, "floats are not accepted")
	_, err = Of(Money{}, usd)
	assert.ErrorIs(t, err, ErrInvalidOperand)
	_, err = Of("1.234", usd)
	assert.ErrorIs(t, err, ErrTooManyDecimals)
	_, err = manager.Of("1", "XYZ")
	assert.ErrorIs(t, err, ErrUnknownCurrency)
}

func TestParityOfMinor(t *testing.T) {
	useConfig(t, laravelConfig())
	usd := MustCurrency("USD")
	manager, err := current()
	require.NoError(t, err)

	for _, minor := range []any{123450, int64(123450), uint32(123450), "123450", " 123450 ", big.NewInt(123450)} {
		m, err := OfMinor(minor, usd)
		require.NoError(t, err)
		assert.Equal(t, "1234.50", m.Decimal())
	}
	m, err := manager.OfMinor(-5, "KWD")
	require.NoError(t, err)
	assert.Equal(t, "-0.005", m.Decimal())

	_, err = OfMinor(10.5, usd)
	assert.ErrorIs(t, err, ErrInvalidOperand)
	assert.ErrorContains(t, err, "floats are not accepted")
	_, err = OfMinor(nil, usd)
	assert.ErrorIs(t, err, ErrInvalidOperand)
}

func TestParityEqualsRejectsInvalidOperands(t *testing.T) {
	useConfig(t, laravelConfig())
	price := of(t, "19.99", "USD")

	assert.True(t, equal(t, price, "19.99"))
	assert.False(t, equal(t, price, of(t, "19.99", "EUR")))

	_, err := price.Equals(19.99)
	assert.ErrorIs(t, err, ErrInvalidOperand)
	_, err = price.Equals("19,99")
	assert.ErrorIs(t, err, ErrInvalidDecimal)
	_, err = price.Equals("19.999")
	assert.ErrorIs(t, err, ErrTooManyDecimals)
}

func TestParityIsSameCurrencyComparesPrecision(t *testing.T) {
	useConfig(t, laravelConfig())
	zeroMMK, err := NewCurrency("MMK", 0)
	require.NoError(t, err)
	iso := MustOf("100", MustCurrency("MMK"))
	custom := MustOf("100", zeroMMK)

	assert.False(t, iso.IsSameCurrency(custom))
	assert.False(t, equal(t, iso, custom))
	_, err = iso.Plus(custom)
	assert.ErrorIs(t, err, ErrCurrencyMismatch)
	_, err = iso.Compare(custom)
	assert.ErrorIs(t, err, ErrCurrencyMismatch)
}

func TestParityRejectsInvalidRoundingModes(t *testing.T) {
	useConfig(t, laravelConfig())
	usd := MustCurrency("USD")
	price := of(t, "12.34", "USD")
	invalid := Rounding(42)

	calls := map[string]func() error{
		"Parse": func() error { _, err := Parse("1.235", usd, invalid); return err },
		"Parse without extra decimals": func() error {
			_, err := Parse("1.23", usd, invalid)
			return err
		},
		"Of":        func() error { _, err := Of("1.235", usd, invalid); return err },
		"Times":     func() error { _, err := price.Times(2, invalid); return err },
		"DividedBy": func() error { _, err := price.DividedBy(2, invalid); return err },
		"Percent":   func() error { _, err := price.Percent(10, invalid); return err },
		"RoundTo":   func() error { _, err := price.RoundTo(0, invalid); return err },
		"RoundTo at the currency's precision": func() error {
			_, err := price.RoundTo(2, invalid)
			return err
		},
		"PercentageOf": func() error {
			_, err := price.PercentageOf(price, 2, invalid)
			return err
		},
		"Avg": func() error { _, err := Avg([]Money{price}, invalid); return err },
	}
	for name, call := range calls {
		t.Run(name, func(t *testing.T) {
			err := call()
			require.ErrorIs(t, err, ErrInvalidOperand)
			assert.ErrorContains(t, err, "rounding(42) is not a rounding mode")
		})
	}
}

func TestParityAllocateMapTies(t *testing.T) {
	useConfig(t, laravelConfig())

	// Two leftover cents, three equal remainders: the first keys in
	// ascending order get them, as in laravel-money's allocate().
	shares, err := AllocateMap(minor(t, "5", ""), map[string]int{
		"platform": 1,
		"agent":    1,
		"owner":    1,
	})
	require.NoError(t, err)
	assert.Equal(t, "0.02", shares["agent"].Decimal())
	assert.Equal(t, "0.02", shares["owner"].Decimal())
	assert.Equal(t, "0.01", shares["platform"].Decimal())

	byID, err := AllocateMap(minor(t, "2", ""), map[int]int{10: 1, 2: 1, 7: 1})
	require.NoError(t, err)
	assert.Equal(t, "0.01", byID[2].Decimal())
	assert.Equal(t, "0.01", byID[7].Decimal())
	assert.Equal(t, "0.00", byID[10].Decimal(), "integer keys sort numerically")
}

func TestParityPercentageOfAndRatioOfNeedAScale(t *testing.T) {
	useConfig(t, laravelConfig())

	percentage, err := of(t, "25", "").PercentageOf(of(t, "200", ""), 2)
	require.NoError(t, err)
	assert.Equal(t, "12.50", percentage)
	ratio, err := of(t, "50", "").RatioOf(of(t, "200", ""), 4)
	require.NoError(t, err)
	assert.Equal(t, "0.2500", ratio)
	ratio, err = of(t, "1", "").RatioOf(of(t, "3", ""), 0, Ceiling)
	require.NoError(t, err)
	assert.Equal(t, "1", ratio)
}

func TestParityStringIsCodeAndDecimal(t *testing.T) {
	cfg := laravelConfig()
	cfg.Locale = "en"
	useConfig(t, cfg)

	price := of(t, "1234.5", "USD")
	assert.Equal(t, "USD 1234.50", price.String())
	assert.Equal(t, "$1,234.50", price.Format())
	assert.Equal(t, "KWD -0.005", of(t, "-0.005", "KWD").String())
}
