package money

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefaultConfig(t *testing.T) {
	assert.Equal(t, Config{
		DefaultCurrency: "USD",
		Rounding:        HalfUp,
		Serialization:   Serialization{Amount: AmountMinor, IncludeDecimal: true, IncludeFormatted: true},
	}, DefaultConfig())
}

func TestNewManager(t *testing.T) {
	manager, err := NewManager(Config{DefaultCurrency: " mmk ", Rounding: Floor, Currencies: map[string]int{"PTS": 0}, Locale: "my_MM"})
	require.NoError(t, err)

	assert.Equal(t, "MMK", manager.DefaultCurrency().Code())
	assert.Equal(t, Floor, manager.Rounding())
	assert.Equal(t, "my_MM", manager.Locale())
	assert.Equal(t, Serialization{Amount: AmountMinor}, manager.Serialization(), "an empty amount format is minor")
	assert.True(t, manager.Registry().Has("PTS"))

	empty, err := NewManager(Config{})
	require.NoError(t, err)
	assert.Equal(t, FallbackCurrency, empty.DefaultCurrency().Code())
}

func TestNewManagerRejectsInvalidConfig(t *testing.T) {
	_, err := NewManager(Config{DefaultCurrency: "NOPE"})
	require.ErrorIs(t, err, ErrUnknownCurrency)
	assert.ErrorContains(t, err, `money: the default currency "NOPE" (config money.default_currency / env MONEY_CURRENCY) is not a known currency`)
	var unknown *UnknownCurrencyError
	require.ErrorAs(t, err, &unknown)
	assert.True(t, unknown.Default)

	_, err = NewManager(Config{Currencies: map[string]int{"": 2}})
	assert.ErrorIs(t, err, ErrInvalidConfig)

	manager, err := NewManager(Config{DefaultCurrency: "PTS", Currencies: map[string]int{"PTS": 0}})
	require.NoError(t, err)
	assert.Equal(t, "PTS", manager.DefaultCurrency().Code(), "a custom currency can be the default")
}

func TestManagerConstructors(t *testing.T) {
	manager, err := NewManager(Config{DefaultCurrency: "USD", Currencies: map[string]int{"MMK": 0}})
	require.NoError(t, err)

	tests := []struct {
		name string
		call func() (Money, error)
		want string
	}{
		{"Parse default", func() (Money, error) { return manager.Parse("1,234.50", "") }, "USD 1234.50"},
		{"Parse rounding", func() (Money, error) { return manager.Parse("1.235", "USD", HalfEven) }, "USD 1.24"},
		{"Parse override", func() (Money, error) { return manager.Parse("1500", "mmk") }, "MMK 1500"},
		{"Make", func() (Money, error) { return manager.Make(1050, "") }, "USD 10.50"},
		{"Make JPY", func() (Money, error) { return manager.Make(1050, "JPY") }, "JPY 1050"},
		{"OfMinor", func() (Money, error) { return manager.OfMinor("123450", "") }, "USD 1234.50"},
		{"Zero", func() (Money, error) { return manager.Zero("KWD") }, "KWD 0.000"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m, err := tt.call()
			require.NoError(t, err)
			assert.Equal(t, tt.want, m.String())
		})
	}

	for name, call := range map[string]func() (Money, error){
		"Parse":   func() (Money, error) { return manager.Parse("1", "XYZ") },
		"Make":    func() (Money, error) { return manager.Make(1, "XYZ") },
		"OfMinor": func() (Money, error) { return manager.OfMinor("1", "XYZ") },
		"Zero":    func() (Money, error) { return manager.Zero("XYZ") },
	} {
		_, err := call()
		assert.ErrorIs(t, err, ErrUnknownCurrency, name)
	}

	precision, err := manager.Precision("mmk")
	require.NoError(t, err)
	assert.Equal(t, 0, precision)
	precision, err = manager.Precision("")
	require.NoError(t, err)
	assert.Equal(t, 2, precision)
	_, err = manager.Precision("XYZ")
	assert.ErrorIs(t, err, ErrUnknownCurrency)

	c, err := manager.Currency("")
	require.NoError(t, err)
	assert.Equal(t, "USD", c.Code())
}

func TestCurrentFallsBackOutsideAnApp(t *testing.T) {
	resetRegistered()
	manager, err := current()
	require.NoError(t, err)
	assert.Equal(t, DefaultConfig().Serialization, manager.Serialization())
	assert.Equal(t, "USD", manager.DefaultCurrency().Code())
	assert.Same(t, manager, fallbackManager())
}
