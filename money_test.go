package money

import (
	"encoding/json"
	"errors"
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDecimal(t *testing.T) {
	tests := []struct {
		amount int64
		code   string
		want   string
	}{
		{1050, "USD", "10.50"},
		{2500, "JPY", "2500"},
		{2500, "MMK", "25.00"},
		{5, "USD", "0.05"},
		{-5, "USD", "-0.05"},
		{0, "USD", "0.00"},
		{0, "JPY", "0"},
		{-2500, "JPY", "-2500"},
		{1, "BHD", "0.001"},
		{12345, "CLF", "1.2345"},
		{math.MaxInt64, "USD", "92233720368547758.07"},
		{math.MinInt64, "USD", "-92233720368547758.08"},
	}
	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			m := New(tt.amount, MustCurrency(tt.code))
			assert.Equal(t, tt.want, m.Decimal())
			assert.Equal(t, tt.want+" "+tt.code, m.String())
		})
	}
}

func TestPredicates(t *testing.T) {
	usd := MustCurrency("USD")
	assert.True(t, New(0, usd).IsZero())
	assert.True(t, New(1, usd).IsPositive())
	assert.True(t, New(-1, usd).IsNegative())
	assert.False(t, New(0, usd).IsPositive())
	assert.False(t, New(0, usd).IsNegative())
	assert.Equal(t, int64(1050), New(1050, usd).Amount())
	assert.Equal(t, "USD", New(1050, usd).Currency().Code())
}

func TestEqualsAndCompare(t *testing.T) {
	usd, eur := MustCurrency("USD"), MustCurrency("EUR")

	assert.True(t, New(100, usd).Equals(New(100, usd)))
	assert.False(t, New(100, usd).Equals(New(101, usd)))
	assert.False(t, New(100, usd).Equals(New(100, eur)))
	assert.True(t, New(1, usd).SameCurrency(New(2, usd)))

	tests := []struct {
		a, b int64
		want int
	}{{1, 2, -1}, {2, 2, 0}, {3, 2, 1}}
	for _, tt := range tests {
		got, err := New(tt.a, usd).Compare(New(tt.b, usd))
		require.NoError(t, err)
		assert.Equal(t, tt.want, got)
	}

	_, err := New(1, usd).Compare(New(1, eur))
	assert.ErrorIs(t, err, ErrCurrencyMismatch)
}

func TestAddAndSubtract(t *testing.T) {
	usd, eur := MustCurrency("USD"), MustCurrency("EUR")

	sum, err := New(1050, usd).Add(New(-50, usd))
	require.NoError(t, err)
	assert.Equal(t, New(1000, usd), sum)

	diff, err := New(1050, usd).Subtract(New(2000, usd))
	require.NoError(t, err)
	assert.Equal(t, New(-950, usd), diff)

	tests := []struct {
		name    string
		op      func() (Money, error)
		wantErr error
	}{
		{"add mismatch", func() (Money, error) { return New(1, usd).Add(New(1, eur)) }, ErrCurrencyMismatch},
		{"subtract mismatch", func() (Money, error) { return New(1, usd).Subtract(New(1, eur)) }, ErrCurrencyMismatch},
		{"add overflow", func() (Money, error) { return New(math.MaxInt64, usd).Add(New(1, usd)) }, ErrOverflow},
		{"add underflow", func() (Money, error) { return New(math.MinInt64, usd).Add(New(-1, usd)) }, ErrOverflow},
		{"subtract overflow", func() (Money, error) { return New(math.MaxInt64, usd).Subtract(New(-1, usd)) }, ErrOverflow},
		{"subtract underflow", func() (Money, error) { return New(math.MinInt64, usd).Subtract(New(1, usd)) }, ErrOverflow},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := tt.op()
			assert.ErrorIs(t, err, tt.wantErr)
		})
	}
}

func TestMoneyJSON(t *testing.T) {
	data, err := json.Marshal(New(1050, MustCurrency("USD")))
	require.NoError(t, err)
	assert.JSONEq(t, `{"amount":"1050","currency":"USD"}`, string(data))

	tests := []struct {
		input string
		want  Money
	}{
		{`{"amount":"1050","currency":"USD"}`, New(1050, MustCurrency("USD"))},
		{`{"amount":2500,"currency":"mmk"}`, New(2500, MustCurrency("MMK"))},
		{`{"amount":"-5","currency":"JPY"}`, New(-5, MustCurrency("JPY"))},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			var m Money
			require.NoError(t, json.Unmarshal([]byte(tt.input), &m))
			assert.Equal(t, tt.want, m)
		})
	}
}

func TestMoneyJSONErrors(t *testing.T) {
	tests := []struct {
		input   string
		wantErr error
	}{
		{`{"amount":"10.50","currency":"USD"}`, ErrInvalidStoredAmount},
		{`{"amount":10.5,"currency":"USD"}`, ErrInvalidStoredAmount},
		{`{"amount":"abc","currency":"USD"}`, ErrInvalidStoredAmount},
		{`{"amount":true,"currency":"USD"}`, ErrInvalidStoredAmount},
		{`{"currency":"USD"}`, ErrInvalidStoredAmount},
		{`{"amount":"1","currency":"XYZ"}`, ErrUnknownCurrency},
		{`{"amount":"1"}`, ErrUnknownCurrency},
		{`[1,2]`, ErrInvalidStoredAmount},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			var m Money
			err := json.Unmarshal([]byte(tt.input), &m)
			assert.ErrorIs(t, err, tt.wantErr)
			var typed *InvalidMoneyError
			assert.True(t, errors.As(err, &typed))
		})
	}
}

func TestInvalidMoneyError(t *testing.T) {
	err := newError(ErrOverflow, "value %d", 5)
	assert.Equal(t, "money: value 5", err.Error())
	assert.Equal(t, ErrOverflow, err.Unwrap())
	assert.ErrorIs(t, err, ErrOverflow)
	assert.NotErrorIs(t, err, ErrInvalidDecimal)
}
