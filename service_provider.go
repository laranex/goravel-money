package money

import (
	"fmt"
	"reflect"
	"sync/atomic"

	"github.com/goravel/framework/contracts/binding"
	contractsconfig "github.com/goravel/framework/contracts/config"
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

// Register binds the *Manager as a singleton, configured from config/money.go
// (see ConfigFrom). Without the published file it uses MONEY_CURRENCY and
// laravel-money's defaults.
func (r *ServiceProvider) Register(app foundation.Application) {
	registeredApp.Store(&appHolder{app: app})

	app.Singleton(Binding, func(app foundation.Application) (any, error) {
		config := app.MakeConfig()
		if config == nil {
			return nil, fmt.Errorf("goravel-money: the config facade is not registered")
		}
		cfg, err := ConfigFrom(config)
		if err != nil {
			return nil, err
		}

		return NewManager(cfg)
	})
}

// ConfigFrom reads the money.* config values: default_currency (else the
// MONEY_CURRENCY env, else USD), rounding, currencies, locale (else
// app.locale) and serialization.
func ConfigFrom(config contractsconfig.Config) (Config, error) {
	cfg := DefaultConfig()
	cfg.DefaultCurrency = config.GetString("money.default_currency", config.EnvString("MONEY_CURRENCY", FallbackCurrency))

	switch rounding := config.Get("money.rounding", HalfUp.String()).(type) {
	case Rounding:
		cfg.Rounding = rounding
	case string:
		mode, err := ParseRounding(rounding)
		if err != nil {
			return Config{}, err
		}
		cfg.Rounding = mode
	case nil:
	default:
		return Config{}, invalidConfig("rounding", "a rounding mode name such as \"half_up\"")
	}

	currencies, err := customCurrencies(config.Get("money.currencies", nil))
	if err != nil {
		return Config{}, err
	}
	cfg.Currencies = currencies

	cfg.Locale = config.GetString("money.locale", "")
	if cfg.Locale == "" {
		cfg.Locale = config.GetString("app.locale", "")
	}

	amount, _ := config.Get("money.serialization.amount", string(AmountMinor)).(string)
	cfg.Serialization = Serialization{
		Amount:           AmountFormat(amount),
		IncludeDecimal:   config.GetBool("money.serialization.include_decimal", true),
		IncludeFormatted: config.GetBool("money.serialization.include_formatted", true),
	}

	return cfg, nil
}

// customCurrencies reads money.currencies, a map of codes to integer decimal places.
func customCurrencies(value any) (map[string]int, error) {
	invalid := invalidConfig("currencies", `a map of currency codes to non-negative integer decimal places, e.g. "PTS": 0`)
	if value == nil {
		return nil, nil
	}
	rv := reflect.ValueOf(value)
	if rv.Kind() != reflect.Map || rv.Type().Key().Kind() != reflect.String {
		return nil, invalid
	}
	out := make(map[string]int, rv.Len())
	iter := rv.MapRange()
	for iter.Next() {
		v := reflect.ValueOf(iter.Value().Interface())
		switch v.Kind() {
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			out[iter.Key().String()] = int(v.Int())
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			out[iter.Key().String()] = int(v.Uint())
		default:
			return nil, invalid
		}
	}

	return out, nil
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
