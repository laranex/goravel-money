package money

import (
	"testing"

	"github.com/goravel/framework/contracts/foundation"
	mocksfoundation "github.com/goravel/framework/mocks/foundation"
	"github.com/stretchr/testify/require"
)

// resetRegistered clears the registered application between tests.
func resetRegistered() { registeredApp.Store(nil) }

// setRegistered registers app without going through ServiceProvider.Register.
func setRegistered(app foundation.Application) { registeredApp.Store(&appHolder{app: app}) }

// useConfig registers a mock application whose manager has cfg, like a
// Goravel app with config/money.go, until the test ends.
func useConfig(t *testing.T, cfg Config) *Manager {
	t.Helper()
	manager, err := NewManager(cfg)
	require.NoError(t, err)
	app := mocksfoundation.NewApplication(t)
	app.EXPECT().Make(Binding).Return(manager, nil).Maybe()
	setRegistered(app)
	t.Cleanup(resetRegistered)

	return manager
}

// useDefaultCurrency registers a manager whose default currency is code.
func useDefaultCurrency(t *testing.T, code string) *Manager {
	t.Helper()
	cfg := DefaultConfig()
	cfg.DefaultCurrency = code

	return useConfig(t, cfg)
}

// laravelConfig is the config of laravel-money's test suite: USD, and a
// custom PTS currency with no decimals.
func laravelConfig() Config {
	cfg := DefaultConfig()
	cfg.Currencies = map[string]int{"PTS": 0}

	return cfg
}

// of parses a decimal amount with the current manager, like Money::of().
func of(t testing.TB, amount, code string, rounding ...Rounding) Money {
	t.Helper()
	manager, err := current()
	require.NoError(t, err)
	m, err := manager.Parse(amount, code, rounding...)
	require.NoError(t, err)

	return m
}

// minor builds Money from minor units with the current manager, like Money::ofMinor().
func minor(t testing.TB, amount, code string) Money {
	t.Helper()
	manager, err := current()
	require.NoError(t, err)
	m, err := manager.OfMinor(amount, code)
	require.NoError(t, err)

	return m
}

func decimals(monies []Money) []string {
	out := make([]string, len(monies))
	for i, m := range monies {
		out[i] = m.Decimal()
	}

	return out
}

func amounts(monies []Money) []string {
	out := make([]string, len(monies))
	for i, m := range monies {
		out[i] = m.Amount()
	}

	return out
}
