package money

import (
	"errors"
	"testing"
	"time"

	"github.com/goravel/framework/contracts/binding"
	"github.com/goravel/framework/contracts/foundation"
	mocksfoundation "github.com/goravel/framework/mocks/foundation"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// registerWith runs ServiceProvider.Register on a mock application and
// returns the singleton callback it bound.
func registerWith(t *testing.T, app *mocksfoundation.Application) func(foundation.Application) (any, error) {
	t.Helper()
	var callback func(foundation.Application) (any, error)
	app.EXPECT().Singleton(Binding, mock.AnythingOfType("func(foundation.Application) (interface {}, error)")).
		Run(func(_ any, cb func(foundation.Application) (any, error)) { callback = cb }).Once()

	(&ServiceProvider{}).Register(app)
	require.NotNil(t, callback)

	return callback
}

func TestServiceProviderRelationship(t *testing.T) {
	rel := (&ServiceProvider{}).Relationship()
	assert.Equal(t, []string{Binding}, rel.Bindings)
	assert.Equal(t, []string{binding.Config}, rel.Dependencies)
}

// fakeConfig is a Goravel config backed by a map, with env values under "env.".
type fakeConfig map[string]any

func (c fakeConfig) Env(name string, def ...any) any {
	if v, ok := c["env."+name]; ok {
		return v
	}

	return first(def)
}

func (c fakeConfig) EnvString(name string, def ...string) string {
	if v, ok := c["env."+name]; ok {
		return v.(string)
	}

	return first(def)
}

func (c fakeConfig) EnvBool(name string, def ...bool) bool { return first(def) }

func (c fakeConfig) Add(name string, configuration any) { c[name] = configuration }

func (c fakeConfig) Get(path string, def ...any) any {
	if v, ok := c[path]; ok {
		return v
	}

	return first(def)
}

func (c fakeConfig) GetString(path string, def ...string) string {
	if v, ok := c[path]; ok {
		return v.(string)
	}

	return first(def)
}

func (c fakeConfig) GetInt(path string, def ...int) int { return first(def) }

func (c fakeConfig) GetBool(path string, def ...bool) bool {
	if v, ok := c[path]; ok {
		return v.(bool)
	}

	return first(def)
}

func (c fakeConfig) GetDuration(path string, def ...time.Duration) time.Duration {
	return first(def)
}

func (c fakeConfig) GetStringSlice(path string, def ...[]string) []string { return first(def) }

func (c fakeConfig) UnmarshalKey(string, any) error { return nil }

func first[T any](values []T) T {
	var zero T
	if len(values) > 0 {
		return values[0]
	}

	return zero
}

func TestConfigFrom(t *testing.T) {
	tests := []struct {
		name   string
		config fakeConfig
		want   Config
	}{
		{"nothing published", fakeConfig{}, DefaultConfig()},
		{"env fallback", fakeConfig{"env.MONEY_CURRENCY": "EUR"}, func() Config { c := DefaultConfig(); c.DefaultCurrency = "EUR"; return c }()},
		{"app locale", fakeConfig{"app.locale": "en"}, func() Config { c := DefaultConfig(); c.Locale = "en"; return c }()},
		{"everything", fakeConfig{
			"env.MONEY_CURRENCY":                    "EUR",
			"money.default_currency":                "MMK",
			"money.rounding":                        "half_even",
			"money.currencies":                      map[string]any{"pts": 0, "gem": int64(4), "mmk": uint(0)},
			"money.locale":                          "my_MM",
			"app.locale":                            "en",
			"money.serialization.amount":            "decimal",
			"money.serialization.include_decimal":   false,
			"money.serialization.include_formatted": false,
		}, Config{
			DefaultCurrency: "MMK",
			Rounding:        HalfEven,
			Currencies:      map[string]int{"pts": 0, "gem": 4, "mmk": 0},
			Locale:          "my_MM",
			Serialization:   Serialization{Amount: AmountDecimal},
		}},
		{"typed values", fakeConfig{"money.rounding": Floor, "money.currencies": map[string]int{"PTS": 0}}, func() Config {
			c := DefaultConfig()
			c.Rounding = Floor
			c.Currencies = map[string]int{"PTS": 0}
			return c
		}()},
		{"null rounding", fakeConfig{"money.rounding": nil}, DefaultConfig()},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ConfigFrom(tt.config)
			require.NoError(t, err)
			if tt.want.Currencies == nil {
				tt.want.Currencies = got.Currencies
				assert.Empty(t, got.Currencies)
			}
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestConfigFromRejectsInvalidValues(t *testing.T) {
	tests := []struct {
		name    string
		config  fakeConfig
		message string
	}{
		{"rounding name", fakeConfig{"money.rounding": "sideways"}, "money.rounding must be one of"},
		{"rounding type", fakeConfig{"money.rounding": 3}, "money.rounding must be a rounding mode name"},
		{"currencies list", fakeConfig{"money.currencies": []string{"PTS"}}, "money.currencies must be a map"},
		{"currencies precision", fakeConfig{"money.currencies": map[string]any{"PTS": "2"}}, "money.currencies must be a map"},
		{"currencies keys", fakeConfig{"money.currencies": map[int]int{1: 2}}, "money.currencies must be a map"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ConfigFrom(tt.config)
			require.ErrorIs(t, err, ErrInvalidConfig)
			assert.ErrorContains(t, err, tt.message)
		})
	}
}

func TestServiceProviderBindsManagerFromConfig(t *testing.T) {
	t.Cleanup(resetRegistered)
	app := mocksfoundation.NewApplication(t)
	callback := registerWith(t, app)

	app.EXPECT().MakeConfig().Return(fakeConfig{
		"money.default_currency": "PTS",
		"money.currencies":       map[string]any{"pts": 0},
		"money.rounding":         "floor",
		"app.locale":             "de_DE",
	}).Once()

	instance, err := callback(app)
	require.NoError(t, err)
	manager, ok := instance.(*Manager)
	require.True(t, ok)
	assert.Equal(t, "PTS", manager.DefaultCurrency().Code())
	assert.Equal(t, Floor, manager.Rounding())
	assert.Equal(t, "de_DE", manager.Locale())
}

func TestServiceProviderRejectsInvalidConfig(t *testing.T) {
	tests := []struct {
		name   string
		config fakeConfig
		err    error
	}{
		{"unknown default currency", fakeConfig{"money.default_currency": "XYZ"}, ErrUnknownCurrency},
		{"invalid rounding", fakeConfig{"money.rounding": "up"}, ErrInvalidConfig},
		{"invalid serialization", fakeConfig{"money.serialization.amount": "cents"}, ErrInvalidConfig},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Cleanup(resetRegistered)
			app := mocksfoundation.NewApplication(t)
			callback := registerWith(t, app)
			app.EXPECT().MakeConfig().Return(tt.config).Once()

			_, err := callback(app)
			assert.ErrorIs(t, err, tt.err)
		})
	}
}

func TestServiceProviderRequiresConfigFacade(t *testing.T) {
	t.Cleanup(resetRegistered)
	app := mocksfoundation.NewApplication(t)
	callback := registerWith(t, app)

	app.EXPECT().MakeConfig().Return(nil).Once()

	_, err := callback(app)
	assert.ErrorContains(t, err, "config facade is not registered")
}

func TestServiceProviderPublishesConfig(t *testing.T) {
	app := mocksfoundation.NewApplication(t)
	app.EXPECT().ConfigPath("money.go").Return("/app/config/money.go").Once()
	app.EXPECT().Publishes(PackageName, map[string]string{
		"config/money.go": "/app/config/money.go",
	}, "goravel-money", "goravel-money-config").Once()

	(&ServiceProvider{}).Boot(app)
}

func TestResolve(t *testing.T) {
	manager, err := NewManager(Config{DefaultCurrency: "MMK"})
	require.NoError(t, err)

	tests := []struct {
		name     string
		instance any
		err      error
		wantErr  string
	}{
		{"bound manager", manager, nil, ""},
		{"container error", nil, errors.New("not bound"), "not bound"},
		{"wrong type", "nope", nil, "not *money.Manager"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := mocksfoundation.NewApplication(t)
			app.EXPECT().Make(Binding).Return(tt.instance, tt.err).Once()

			got, err := Resolve(app)
			if tt.wantErr != "" {
				assert.ErrorContains(t, err, tt.wantErr)

				return
			}
			require.NoError(t, err)
			assert.Same(t, manager, got)
		})
	}
}

func TestRegistered(t *testing.T) {
	resetRegistered()
	t.Cleanup(resetRegistered)

	_, err := Registered()
	assert.ErrorContains(t, err, "register &money.ServiceProvider{}")

	manager, err := NewManager(Config{DefaultCurrency: "EUR"})
	require.NoError(t, err)
	app := mocksfoundation.NewApplication(t)
	app.EXPECT().Make(Binding).Return(manager, nil).Once()
	setRegistered(app)

	got, err := Registered()
	require.NoError(t, err)
	assert.Same(t, manager, got)
}
