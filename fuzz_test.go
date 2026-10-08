package money

import (
	"math"
	"math/big"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func fuzzCurrency(selector int) Currency {
	switch ((selector % 4) + 4) % 4 {
	case 0:
		return MustCurrency("JPY")
	case 1:
		return MustCurrency("USD")
	case 2:
		return MustCurrency("KWD")
	default:
		return MustCurrency("CLF")
	}
}

func FuzzDecimalRoundTrip(f *testing.F) {
	for _, seed := range []string{"0", "1", "-1", "1050", "-1050", "9223372036854775807", "-9223372036854775808", "123456789012345678901234567890"} {
		f.Add(seed, 1)
	}
	f.Fuzz(func(t *testing.T, digits string, selector int) {
		currency := fuzzCurrency(selector)
		m, err := OfMinor(digits, currency)
		if err != nil {
			return
		}
		parsed, err := Parse(m.Decimal(), currency)
		require.NoError(t, err)
		assert.Equal(t, m, parsed, "%s round-tripped to %s", m, parsed)

		formatted := m.Format("en_IN")
		assert.NotEmpty(t, formatted)
	})
}

func FuzzParseNeverPanics(f *testing.F) {
	for _, seed := range []string{"", "1,234.50", "-0.005", "12 34", "1e3", "..", "+-1", "1\u00A0234"} {
		f.Add(seed, 1, uint8(0))
	}
	f.Fuzz(func(t *testing.T, input string, selector int, mode uint8) {
		currency := fuzzCurrency(selector)
		rounding := Rounding(int(mode) % len(roundingNames))
		m, err := Parse(input, currency, rounding)
		if err != nil {
			assert.ErrorIs(t, err, ErrMoney)

			return
		}
		strict, err := Parse(input, currency)
		if err == nil {
			assert.Equal(t, m, strict, "rounding only matters for extra decimals")
		}
	})
}

func FuzzAllocateKeepsEveryMinorUnit(f *testing.F) {
	f.Add(int64(1001), uint16(3), uint16(5), uint16(11))
	f.Add(int64(-100), uint16(1), uint16(1), uint16(1))
	f.Add(int64(math.MaxInt64), uint16(0), uint16(7), uint16(65535))
	f.Fuzz(func(t *testing.T, amount int64, a, b, c uint16) {
		m := New(amount, MustCurrency("USD"))
		shares, err := m.Allocate(a, b, c)
		if a == 0 && b == 0 && c == 0 {
			assert.ErrorIs(t, err, ErrInvalidAllocation)

			return
		}
		require.NoError(t, err)
		total, err := Sum(shares)
		require.NoError(t, err)
		assert.True(t, m.Equals(total))

		ratios := []uint16{a, b, c}
		sum := new(big.Int)
		for _, r := range ratios {
			sum.Add(sum, big.NewInt(int64(r)))
		}
		for i, share := range shares {
			// Each share is within one minor unit of its exact proportion.
			exact := new(big.Int).Mul(m.BigInt(), big.NewInt(int64(ratios[i])))
			diff := new(big.Int).Sub(new(big.Int).Mul(share.BigInt(), sum), exact)
			assert.LessOrEqual(t, new(big.Int).Abs(diff).Cmp(sum), 0, "share %d", i)
		}
	})
}

func TestDivideRoundsExactly(t *testing.T) {
	tests := []struct {
		num, den int64
		rounding Rounding
		want     int64
	}{
		{7, 2, HalfUp, 4},
		{-7, 2, HalfUp, -4},
		{7, -2, HalfUp, -4},
		{-7, -2, HalfDown, 3},
		{5, 2, HalfEven, 2},
		{7, 2, HalfEven, 4},
		{5, 2, HalfOdd, 3},
		{7, 3, Floor, 2},
		{-7, 3, Floor, -3},
		{7, 3, Ceiling, 3},
		{-7, 3, Ceiling, -2},
		{6, 3, Floor, 2},
	}
	for _, tt := range tests {
		got, err := divide(big.NewInt(tt.num), big.NewInt(tt.den), tt.rounding)
		require.NoError(t, err)
		assert.Equal(t, tt.want, got.Int64(), "%d/%d %s", tt.num, tt.den, tt.rounding)
	}

	_, err := divide(big.NewInt(1), big.NewInt(0), HalfUp)
	assert.ErrorIs(t, err, ErrDivisionByZero)
}
