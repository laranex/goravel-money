package money

import (
	"sync"
)

// FallbackCurrency is used when the money.default_currency config value and
// the MONEY_CURRENCY environment variable are both unset.
const FallbackCurrency = "USD"

// AmountFormat is how the amount is serialized: in minor units or as a decimal.
type AmountFormat string

const (
	// AmountMinor serializes the amount in minor units, "123450". The default.
	AmountMinor AmountFormat = "minor"
	// AmountDecimal serializes the amount as a decimal, "1234.50".
	AmountDecimal AmountFormat = "decimal"
)

// Serialization is the JSON shape of Money (config money.serialization).
// Amounts are always strings.
type Serialization struct {
	// Amount is AmountMinor ("123450", the default) or AmountDecimal ("1234.50").
	Amount AmountFormat
	// IncludeDecimal adds a "decimal" key when Amount is AmountMinor.
	IncludeDecimal bool
	// IncludeFormatted adds a "formatted" key ("$1,234.50").
	IncludeFormatted bool
}

// Config configures a Manager. Start from DefaultConfig.
type Config struct {
	// DefaultCurrency is used whenever no currency code is given; empty means FallbackCurrency.
	DefaultCurrency string
	// Rounding is the default rounding mode (the zero value is HalfUp).
	Rounding Rounding
	// Currencies are custom currencies as code => decimal places; they
	// override ISO 4217 precision, e.g. {"PTS": 0}.
	Currencies map[string]int
	// Locale is used by Format and the "formatted" key; empty formats as "USD 1234.50".
	Locale string
	// Serialization is the JSON shape of Money.
	Serialization Serialization
}

// DefaultConfig returns laravel-money's defaults: USD, HalfUp, no custom
// currencies, no locale, and JSON with the amount in minor units plus the
// "decimal" and "formatted" keys.
func DefaultConfig() Config {
	return Config{
		DefaultCurrency: FallbackCurrency,
		Rounding:        HalfUp,
		Serialization:   Serialization{Amount: AmountMinor, IncludeDecimal: true, IncludeFormatted: true},
	}
}

// Manager creates, parses and formats Money with the configured default
// currency, rounding, custom currencies, locale and serialization. The service
// provider binds one into the container; resolve it with facades.Money(). It
// is immutable and safe for concurrent use.
type Manager struct {
	registry        *Registry
	defaultCurrency Currency
	rounding        Rounding
	locale          string
	serialization   Serialization
}

// NewManager validates cfg and returns a Manager.
func NewManager(cfg Config) (*Manager, error) {
	registry, err := NewRegistry(cfg.Currencies)
	if err != nil {
		return nil, err
	}
	code := normalizeCode(cfg.DefaultCurrency)
	if code == "" {
		code = FallbackCurrency
	}
	defaultCurrency, err := registry.Lookup(code)
	if err != nil {
		return nil, &UnknownCurrencyError{Code: code, Default: true}
	}
	if cfg.Rounding < HalfUp || cfg.Rounding > Floor {
		return nil, invalidConfig("rounding", "a valid rounding mode")
	}
	serialization := cfg.Serialization
	switch serialization.Amount {
	case "":
		serialization.Amount = AmountMinor
	case AmountMinor, AmountDecimal:
	default:
		return nil, invalidConfig("serialization.amount", `"minor" or "decimal"`)
	}

	return &Manager{
		registry:        registry,
		defaultCurrency: defaultCurrency,
		rounding:        cfg.Rounding,
		locale:          cfg.Locale,
		serialization:   serialization,
	}, nil
}

// Registry returns the currency registry (ISO 4217 plus custom currencies).
func (m *Manager) Registry() *Registry { return m.registry }

// DefaultCurrency returns the currency used when none is given.
func (m *Manager) DefaultCurrency() Currency { return m.defaultCurrency }

// Rounding returns the default rounding mode (money.rounding).
func (m *Manager) Rounding() Rounding { return m.rounding }

// Locale returns the formatting locale (money.locale, else app.locale).
func (m *Manager) Locale() string { return m.locale }

// Serialization returns the JSON shape of Money (money.serialization).
func (m *Manager) Serialization() Serialization { return m.serialization }

// Currency resolves a currency code with the registry; an empty code returns
// the default currency.
func (m *Manager) Currency(code string) (Currency, error) {
	if normalizeCode(code) == "" {
		return m.defaultCurrency, nil
	}

	return m.registry.Lookup(code)
}

// Precision returns the number of decimals of a currency (empty code: the default).
func (m *Manager) Precision(code string) (int, error) {
	c, err := m.Currency(code)

	return c.minorUnits, err
}

// Parse returns Money for a decimal amount such as "1,234.50" in the given
// currency (empty code: the default currency). See the package-level Parse
// for the strict parsing rules.
func (m *Manager) Parse(amount, code string, rounding ...Rounding) (Money, error) {
	c, err := m.Currency(code)
	if err != nil {
		return Money{}, err
	}

	return Parse(amount, c, rounding...)
}

// Make returns Money for an amount in minor units: Make(1050, "") is 10.50
// in the default currency, Make(1050, "JPY") is 1,050 JPY.
func (m *Manager) Make(minor int64, code string) (Money, error) {
	c, err := m.Currency(code)
	if err != nil {
		return Money{}, err
	}

	return New(minor, c), nil
}

// OfMinor returns Money for an amount in minor units given as an integer
// string of any size, such as "123450".
func (m *Manager) OfMinor(minor, code string) (Money, error) {
	c, err := m.Currency(code)
	if err != nil {
		return Money{}, err
	}

	return OfMinor(minor, c)
}

// Zero returns zero in the given currency (empty code: the default).
func (m *Manager) Zero(code string) (Money, error) {
	c, err := m.Currency(code)
	if err != nil {
		return Money{}, err
	}

	return Zero(c), nil
}

// Format formats money for display in the given locale, or in the configured
// locale when none is given: "$1,234.50" for en, "1.234,50 €" for de_DE. An
// empty or unsupported locale returns "USD 1234.50". See FormatDecimal.
func (m *Manager) Format(money Money, locale ...string) string {
	l := m.locale
	if len(locale) > 0 {
		l = locale[0]
	}

	return FormatDecimal(money.Decimal(), money.currency.code, l)
}

var fallbackManager = sync.OnceValue(func() *Manager {
	manager, err := NewManager(DefaultConfig())
	if err != nil {
		panic(err)
	}

	return manager
})

// current returns the registered application's manager, or a manager with
// DefaultConfig outside a Goravel application.
func current() (*Manager, error) {
	if registeredApp.Load() == nil {
		return fallbackManager(), nil
	}

	return Registered()
}
