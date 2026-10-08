package money

import (
	"fmt"
	"sync/atomic"

	"github.com/goravel/framework/contracts/binding"
	"github.com/goravel/framework/contracts/foundation"
)

const (
	// Binding is the container key of the *Manager.
	Binding = "laranex.money"
	// PackageName is the name used by `./artisan vendor:publish --package=...`.
	PackageName = "github.com/laranex/goravel-money/v4"
)

type appHolder struct{ app foundation.Application }

var registeredApp atomic.Pointer[appHolder]

// ServiceProvider registers the money manager with a Goravel application.
// Add &money.ServiceProvider{} to bootstrap/providers.go.
type ServiceProvider struct{}

// Relationship declares the binding and its dependency on the config facade.
func (r *ServiceProvider) Relationship() binding.Relationship {
	return binding.Relationship{
		Bindings:     []string{Binding},
		Dependencies: []string{binding.Config},
	}
}

// Register binds the *Manager as a singleton. Its default currency is the
// money.default_currency config value, falling back to the MONEY_CURRENCY
// environment variable and then to USD when the config file is not published.
func (r *ServiceProvider) Register(app foundation.Application) {
	registeredApp.Store(&appHolder{app: app})

	app.Singleton(Binding, func(app foundation.Application) (any, error) {
		config := app.MakeConfig()
		if config == nil {
			return nil, fmt.Errorf("goravel-money: the config facade is not registered")
		}
		code := config.GetString("money.default_currency", config.EnvString("MONEY_CURRENCY", FallbackCurrency))

		return NewManager(code)
	})
}

// Boot registers config/money.go for `./artisan vendor:publish`
// (--package=github.com/laranex/goravel-money/v4 or --tag=goravel-money-config).
func (r *ServiceProvider) Boot(app foundation.Application) {
	app.Publishes(PackageName, map[string]string{
		"config/money.go": app.ConfigPath("money.go"),
	}, "goravel-money", "goravel-money-config")
}

// Resolve returns the *Manager bound in app's container.
func Resolve(app foundation.Application) (*Manager, error) {
	instance, err := app.Make(Binding)
	if err != nil {
		return nil, err
	}
	manager, ok := instance.(*Manager)
	if !ok {
		return nil, fmt.Errorf("goravel-money: binding %q is %T, not *money.Manager", Binding, instance)
	}

	return manager, nil
}

// Registered returns the *Manager of the application the ServiceProvider was
// registered with. It returns an error when the provider is not registered.
func Registered() (*Manager, error) {
	holder := registeredApp.Load()
	if holder == nil {
		return nil, fmt.Errorf("goravel-money: register &money.ServiceProvider{} in bootstrap/providers.go")
	}

	return Resolve(holder.app)
}

// defaultCurrency is the currency of Column[Default]: the registered
// application's default currency, or FallbackCurrency outside a Goravel app.
func defaultCurrency() (Currency, error) {
	if registeredApp.Load() == nil {
		return LookupCurrency(FallbackCurrency)
	}
	manager, err := Registered()
	if err != nil {
		return Currency{}, err
	}

	return manager.DefaultCurrency(), nil
}
