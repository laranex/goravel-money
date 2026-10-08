package money

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestEveryErrorMatchesErrMoney(t *testing.T) {
	tests := []struct {
		name   string
		err    error
		reason error
	}{
		{"parse", &ParseError{Input: "x", Err: ErrInvalidDecimal}, ErrInvalidDecimal},
		{"decimals", &ParseError{Input: "1.234", Currency: "USD", Precision: 2, Decimals: 3, Err: ErrTooManyDecimals}, ErrTooManyDecimals},
		{"unknown", &UnknownCurrencyError{Code: "XYZ"}, ErrUnknownCurrency},
		{"mismatch", &CurrencyMismatchError{Expected: "USD", Given: New(1, MustCurrency("EUR"))}, ErrCurrencyMismatch},
		{"invalid", divisionByZero(), ErrDivisionByZero},
		{"config", invalidConfig("locale", "a string"), ErrInvalidConfig},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.ErrorIs(t, tt.err, ErrMoney)
			assert.ErrorIs(t, tt.err, tt.reason)
			assert.NotErrorIs(t, tt.err, errors.New("other"))
			assert.Contains(t, tt.err.Error(), "money: ")
		})
	}
}

func TestCurrencyMismatchMessages(t *testing.T) {
	given := New(100, MustCurrency("USD"))
	left := New(500, MustCurrency("EUR"))

	assert.EqualError(t, &CurrencyMismatchError{Left: &left, Expected: "EUR", Given: given},
		"money: cannot combine EUR 5.00 with USD 1.00: the amounts are in different currencies; convert one of them first")
	assert.EqualError(t, &CurrencyMismatchError{Expected: "MMK", Given: given},
		"money: the column stores MMK amounts, but USD 1.00 was given; convert it to MMK first")
	assert.EqualError(t, &CurrencyMismatchError{Expected: "MMK", Given: given, CurrencyColumn: true},
		"money: the currency column holds MMK, but USD 1.00 was given; convert the amount, or change the currency column first")
}

func TestErrorsAreDistinguishableWithErrorsAs(t *testing.T) {
	_, err := Parse("1.234", MustCurrency("USD"))
	var parseErr *ParseError
	var invalid *InvalidMoneyError
	assert.ErrorAs(t, err, &parseErr)
	assert.False(t, errors.As(err, &invalid))
}
