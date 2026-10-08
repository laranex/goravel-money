package money

import (
	"errors"
	"sort"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/laranex/goravel-money/v4/iso"
)

func TestLookupCurrency(t *testing.T) {
	tests := []struct {
		code        string
		want        string
		name        string
		minorUnits  int
		numericCode int
	}{
		{"USD", "USD", "US Dollar", 2, 840},
		{"MMK", "MMK", "Kyat", 2, 104},
		{"JPY", "JPY", "Yen", 0, 392},
		{"BHD", "BHD", "Bahraini Dinar", 3, 48},
		{"CLF", "CLF", "Unidad de Fomento", 4, 990},
		{"eur", "EUR", "Euro", 2, 978},
		{" thb ", "THB", "Baht", 2, 764},
	}
	for _, tt := range tests {
		t.Run(tt.code, func(t *testing.T) {
			c, err := LookupCurrency(tt.code)
			require.NoError(t, err)
			assert.Equal(t, tt.want, c.Code())
			assert.Equal(t, tt.want, c.String())
			assert.Equal(t, tt.name, c.Name())
			assert.Equal(t, tt.minorUnits, c.MinorUnits())
			assert.Equal(t, tt.numericCode, c.NumericCode())
			assert.False(t, c.IsZero())
		})
	}
}

func TestLookupCurrencyRejectsUnknownCodes(t *testing.T) {
	for _, code := range []string{"", "XYZ", "US", "USDD", "€"} {
		t.Run(code, func(t *testing.T) {
			c, err := LookupCurrency(code)
			assert.True(t, c.IsZero())
			assert.ErrorIs(t, err, ErrUnknownCurrency)
			var typed *InvalidMoneyError
			require.True(t, errors.As(err, &typed))
			assert.Contains(t, typed.Error(), "unknown ISO 4217 currency")
		})
	}
}

func TestMustCurrency(t *testing.T) {
	assert.Equal(t, "USD", MustCurrency("usd").Code())
	assert.Panics(t, func() { MustCurrency("XYZ") })
}

func TestCurrencies(t *testing.T) {
	list := Currencies()
	assert.Len(t, list, 179)
	assert.True(t, sort.SliceIsSorted(list, func(i, j int) bool { return list[i].Code() < list[j].Code() }))
	for _, c := range list {
		again, err := LookupCurrency(c.Code())
		require.NoError(t, err)
		assert.True(t, c.Equals(again))
		assert.GreaterOrEqual(t, c.MinorUnits(), 0)
	}
}

func TestCurrencyEquals(t *testing.T) {
	assert.True(t, MustCurrency("USD").Equals(MustCurrency("usd")))
	assert.False(t, MustCurrency("USD").Equals(MustCurrency("EUR")))
	assert.True(t, Currency{}.IsZero())
}

func TestCurrencyText(t *testing.T) {
	text, err := MustCurrency("MMK").MarshalText()
	require.NoError(t, err)
	assert.Equal(t, "MMK", string(text))

	var c Currency
	require.NoError(t, c.UnmarshalText([]byte("jpy")))
	assert.Equal(t, "JPY", c.Code())

	assert.ErrorIs(t, c.UnmarshalText([]byte("XYZ")), ErrUnknownCurrency)
	assert.Equal(t, "JPY", c.Code(), "a failed decode leaves the value unchanged")
}

func TestISOMarkersMatchTheCurrencyTable(t *testing.T) {
	markers := []CurrencyCode{iso.USD{}, iso.MMK{}, iso.JPY{}, iso.EUR{}, iso.THB{}, iso.BHD{}, iso.CLF{}, iso.SGD{}}
	for _, marker := range markers {
		t.Run(marker.CurrencyCode(), func(t *testing.T) {
			c, err := LookupCurrency(marker.CurrencyCode())
			require.NoError(t, err)
			assert.Equal(t, marker.CurrencyCode(), c.Code())
		})
	}
	assert.Equal(t, "", Default{}.CurrencyCode())
}
