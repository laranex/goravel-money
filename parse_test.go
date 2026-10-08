package money

import (
	"errors"
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Vectors produced by moneyphp's DecimalMoneyParser (laravel-money's parser).
func TestParseMatchesLaravelMoney(t *testing.T) {
	tests := []struct {
		input string
		code  string
		want  int64
	}{
		{"10.50", "USD", 1050},
		{"2500", "JPY", 2500},
		{"25.00", "MMK", 2500},
		{"0.99", "EUR", 99},
		{"10.555", "USD", 1056},
		{"10.554", "USD", 1055},
		{"-10.555", "USD", -1056},
		{"0.995", "USD", 100},
		{"9.999", "USD", 1000},
		{".5", "USD", 50},
		{"5.", "USD", 500},
		{"  3.1 ", "USD", 310},
		{"-0.001", "USD", 0},
		{"", "USD", 0},
		{"-.5", "USD", -50},
		{"-.5", "JPY", -1},
		{"-.5", "BHD", -500},
		{"0.5", "JPY", 1},
		{"0.5", "BHD", 500},
		{"2.5", "JPY", 3},
		{"007.10", "USD", 710},
		{"007.10", "JPY", 7},
		{"007.10", "BHD", 7100},
		{"1.2345", "CLF", 12345},
		{"0", "USD", 0},
		{"-0", "USD", 0},
	}
	for _, tt := range tests {
		t.Run(tt.code+"/"+tt.input, func(t *testing.T) {
			m, err := Parse(tt.input, MustCurrency(tt.code))
			require.NoError(t, err)
			assert.Equal(t, tt.want, m.Amount())
			assert.Equal(t, tt.code, m.Currency().Code())
		})
	}
}

func TestParseRejectsInvalidDecimals(t *testing.T) {
	for _, input := range []string{".", "-", "-.", "+5", "1,000", "1e3", "1.2.3", "abc", "10 USD", "$10", "--1", "1-", "0x10", "١٢"} {
		t.Run(input, func(t *testing.T) {
			_, err := Parse(input, MustCurrency("USD"))
			assert.ErrorIs(t, err, ErrInvalidDecimal)
			var typed *InvalidMoneyError
			assert.True(t, errors.As(err, &typed))
		})
	}
}

func TestParseRange(t *testing.T) {
	usd := MustCurrency("USD")

	tests := []struct {
		input   string
		want    int64
		wantErr error
	}{
		{"92233720368547758.07", math.MaxInt64, nil},
		{"-92233720368547758.08", math.MinInt64, nil},
		{"92233720368547758.08", 0, ErrOverflow},
		{"-92233720368547758.09", 0, ErrOverflow},
		{"92233720368547758.075", 0, ErrOverflow},
		{"99999999999999999.99", 0, ErrOverflow},
		{"184467440737095516.15", 0, ErrOverflow},
		{"184467440737095516.155", 0, ErrOverflow},
		{"1000000000000000000000000", 0, ErrOverflow},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			m, err := Parse(tt.input, usd)
			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)

				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, m.Amount())
		})
	}
}

func FuzzParseFormatRoundTrip(f *testing.F) {
	for _, seed := range []int64{0, 1, -1, 1050, -1050, math.MaxInt64, math.MinInt64} {
		f.Add(seed, 2)
	}
	f.Fuzz(func(t *testing.T, amount int64, minor int) {
		currency := MustCurrency("USD")
		switch ((minor % 4) + 4) % 4 {
		case 0:
			currency = MustCurrency("JPY")
		case 3:
			currency = MustCurrency("BHD")
		}
		m := New(amount, currency)
		parsed, err := Parse(m.Decimal(), currency)
		require.NoError(t, err)
		assert.True(t, m.Equals(parsed), "%s round-tripped to %s", m, parsed)
	})
}
