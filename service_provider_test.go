package money

import (
	"errors"
	"testing"

	"github.com/goravel/framework/contracts/binding"
	"github.com/goravel/framework/contracts/foundation"
	mocksconfig "github.com/goravel/framework/mocks/config"
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

func TestServiceProviderBindsManagerFromConfig(t *testing.T) {
	t.Cleanup(resetRegistered)

	tests := []struct {
		name       string
		env        string
		configured string
		want       string
	}{
		{"config value", "USD", "MMK", "MMK"},
		{"env fallback", "EUR", "EUR", "EUR"},
		{"usd fallback", "USD", "USD", "USD"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := mocksfoundation.NewApplication(t)
			config := mocksconfig.NewConfig(t)
			callback := registerWith(t, app)

			app.EXPECT().MakeConfig().Return(config).Once()
			config.EXPECT().EnvString("MONEY_CURRENCY", FallbackCurrency).Return(tt.env).Once()
			config.EXPECT().GetString("money.default_currency", tt.env).Return(tt.configured).Once()

			instance, err := callback(app)
			require.NoError(t, err)
			manager, ok := instance.(*Manager)
			require.True(t, ok)
			assert.Equal(t, tt.want, manager.DefaultCurrency().Code())
		})
	}
}

func TestServiceProviderRejectsInvalidConfig(t *testing.T) {
	t.Cleanup(resetRegistered)
	app := mocksfoundation.NewApplication(t)
	config := mocksconfig.NewConfig(t)
	callback := registerWith(t, app)

	app.EXPECT().MakeConfig().Return(config).Once()
	config.EXPECT().EnvString("MONEY_CURRENCY", FallbackCurrency).Return("USD").Once()
	config.EXPECT().GetString("money.default_currency", "USD").Return("XYZ").Once()

	_, err := callback(app)
	assert.ErrorIs(t, err, ErrUnknownCurrency)
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
	manager, err := NewManager("MMK")
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

	manager, err := NewManager("EUR")
	require.NoError(t, err)
	app := mocksfoundation.NewApplication(t)
	app.EXPECT().Make(Binding).Return(manager, nil).Once()
	setRegistered(app)

	got, err := Registered()
	require.NoError(t, err)
	assert.Same(t, manager, got)
}
