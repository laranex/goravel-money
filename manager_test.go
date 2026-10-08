package money

import (
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewManager(t *testing.T) {
	tests := []struct {
		code string
		want string
	}{{"", "USD"}, {"USD", "USD"}, {"mmk", "MMK"}, {"EUR", "EUR"}}
	for _, tt := range tests {
		t.Run(tt.code, func(t *testing.T) {
			m, err := NewManager(tt.code)
			require.NoError(t, err)
			assert.Equal(t, tt.want, m.DefaultCurrency().Code())
		})
	}

	_, err := NewManager("XYZ")
	assert.ErrorIs(t, err, ErrUnknownCurrency)
}

func TestManagerCurrency(t *testing.T) {
	m, err := NewManager("USD")
	require.NoError(t, err)

	tests := []struct {
		code string
		want string
	}{{"", "USD"}, {"MMK", "MMK"}, {"eur", "EUR"}}
	for _, tt := range tests {
		got, err := m.Currency(tt.code)
		require.NoError(t, err)
		assert.Equal(t, tt.want, got.Code())
	}

	_, err = m.Currency("XYZ")
	assert.ErrorIs(t, err, ErrUnknownCurrency)
}

func TestManagerMake(t *testing.T) {
	m, err := NewManager("USD")
	require.NoError(t, err)

	tests := []struct {
		amount int64
		code   string
		want   Money
	}{
		{1050, "", New(1050, MustCurrency("USD"))},
		{1050, "MMK", New(1050, MustCurrency("MMK"))},
		{1, "EUR", New(1, MustCurrency("EUR"))},
		{-7, "JPY", New(-7, MustCurrency("JPY"))},
	}
	for _, tt := range tests {
		got, err := m.Make(tt.amount, tt.code)
		require.NoError(t, err)
		assert.Equal(t, tt.want, got)
	}

	_, err = m.Make(1, "XYZ")
	assert.ErrorIs(t, err, ErrUnknownCurrency)
}

func TestManagerParseAndFormat(t *testing.T) {
	m, err := NewManager("USD")
	require.NoError(t, err)

	tests := []struct {
		input string
		code  string
		want  Money
		out   string
	}{
		{"10.50", "", New(1050, MustCurrency("USD")), "10.50"},
		{"2500", "JPY", New(2500, MustCurrency("JPY")), "2500"},
		{"25.00", "MMK", New(2500, MustCurrency("MMK")), "25.00"},
		{"0.99", "EUR", New(99, MustCurrency("EUR")), "0.99"},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got, err := m.Parse(tt.input, tt.code)
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
			assert.Equal(t, tt.out, m.Format(got))
		})
	}

	_, err = m.Parse("10", "XYZ")
	assert.ErrorIs(t, err, ErrUnknownCurrency)
	_, err = m.Parse("ten", "")
	assert.ErrorIs(t, err, ErrInvalidDecimal)
}

func TestManagerWithAnotherDefault(t *testing.T) {
	m, err := NewManager("EUR")
	require.NoError(t, err)

	made, err := m.Make(500, "")
	require.NoError(t, err)
	assert.Equal(t, New(500, MustCurrency("EUR")), made)

	parsed, err := m.Parse("5.00", "")
	require.NoError(t, err)
	assert.Equal(t, "5.00", m.Format(parsed))
}

func TestManagerIsSafeForConcurrentUse(t *testing.T) {
	m, err := NewManager("MMK")
	require.NoError(t, err)

	var wg sync.WaitGroup
	for i := 0; i < 32; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			made, err := m.Make(int64(i), "")
			assert.NoError(t, err)
			parsed, err := m.Parse(m.Format(made), "")
			assert.NoError(t, err)
			assert.True(t, made.Equals(parsed))
		}(i)
	}
	wg.Wait()
}
