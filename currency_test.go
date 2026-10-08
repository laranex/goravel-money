package money

import (
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
		{"KWD", "KWD", "Kuwaiti Dinar", 3, 414},
		{"BHD", "BHD", "Bahraini Dinar", 3, 48},
		{"CLP", "CLP", "Chilean Peso", 0, 152},
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
	for _, code := range []string{"", "XYZ", "US", "USDD", "€", "PTS"} {
		t.Run(code, func(t *testing.T) {
			c, err := LookupCurrency(code)
			assert.True(t, c.IsZero())
			assert.ErrorIs(t, err, ErrUnknownCurrency)
			assert.ErrorIs(t, err, ErrMoney)
			var typed *UnknownCurrencyError
			require.ErrorAs(t, err, &typed)
			assert.False(t, typed.Default)
		})
	}

	_, err := LookupCurrency("xyz")
	assert.EqualError(t, err, `money: unknown currency "XYZ"; use an ISO 4217 code (USD, EUR, JPY, MMK...) or register a custom currency in config/money.go under "currencies", e.g. "XYZ": 2`)
	_, err = LookupCurrency(" ")
	assert.ErrorContains(t, err, `e.g. "PTS": 2`)
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
	zeroMMK, err := NewCurrency("MMK", 0)
	require.NoError(t, err)
	assert.False(t, MustCurrency("MMK").Equals(zeroMMK), "a different precision is a different currency")
	assert.True(t, Currency{}.IsZero())
}

func TestNewCurrency(t *testing.T) {
	pts, err := NewCurrency(" pts ", 0)
	require.NoError(t, err)
	assert.Equal(t, "PTS", pts.Code())
	assert.Equal(t, "PTS", pts.Name())
	assert.Equal(t, 0, pts.MinorUnits())
	assert.Equal(t, 0, pts.NumericCode())

	mmk, err := NewCurrency("MMK", 0)
	require.NoError(t, err)
	assert.Equal(t, "Kyat", mmk.Name(), "an ISO code keeps its name")
	assert.Equal(t, 104, mmk.NumericCode())
	assert.Equal(t, 0, mmk.MinorUnits())

	for _, tt := range []struct {
		code  string
		units int
	}{{"", 2}, {"  ", 2}, {"PTS", -1}} {
		_, err := NewCurrency(tt.code, tt.units)
		require.ErrorIs(t, err, ErrInvalidConfig)
		assert.ErrorContains(t, err, "money.currencies must be a map of currency codes to non-negative integer decimal places")
	}
}

func TestRegistry(t *testing.T) {
	registry, err := NewRegistry(map[string]int{"pts": 0, "MMK": 0, " GEM ": 4})
	require.NoError(t, err)

	tests := []struct {
		code      string
		precision int
	}{{"PTS", 0}, {"mmk", 0}, {"GEM", 4}, {"USD", 2}, {"KWD", 3}, {"JPY", 0}}
	for _, tt := range tests {
		c, err := registry.Lookup(tt.code)
		require.NoError(t, err, tt.code)
		assert.Equal(t, tt.precision, c.MinorUnits(), tt.code)
	}
	assert.Equal(t, map[string]int{"PTS": 0, "MMK": 0, "GEM": 4}, registry.Custom())

	assert.True(t, registry.Has("usd"))
	assert.True(t, registry.Has("pts"))
	assert.False(t, registry.Has("XYZ"))
	assert.False(t, registry.Has(""))
	_, err = registry.Lookup("XYZ")
	assert.ErrorIs(t, err, ErrUnknownCurrency)

	empty, err := NewRegistry(nil)
	require.NoError(t, err)
	assert.Empty(t, empty.Custom())
	assert.False(t, empty.Has("PTS"))

	_, err = NewRegistry(map[string]int{"PTS": -1})
	assert.ErrorIs(t, err, ErrInvalidConfig)
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

	useConfig(t, laravelConfig())
	require.NoError(t, c.UnmarshalText([]byte("pts")))
	assert.Equal(t, "PTS", c.Code(), "custom currencies come from the registry")
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
