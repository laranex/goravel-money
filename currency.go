package money

import (
	"sort"
	"strings"
)

// Currency is an ISO 4217 or custom currency with its precision (the number
// of decimal places of its minor unit).
//
// The zero Currency is not a valid currency; obtain one with LookupCurrency,
// MustCurrency, NewCurrency, a Registry or a Manager. Currency is comparable:
// two currencies are equal when their code, name, numeric code and precision
// match.
type Currency struct {
	code        string
	name        string
	minorUnits  int
	numericCode int
}

// LookupCurrency returns the ISO 4217 currency for an alphabetic code such as
// "USD" or "mmk". Unknown codes return an *UnknownCurrencyError. Custom
// currencies from config are resolved by the Manager (facades.Money()) and a
// Registry, not by this function.
func LookupCurrency(code string) (Currency, error) {
	normalized := normalizeCode(code)
	c, ok := isoCurrencies[normalized]
	if !ok {
		return Currency{}, &UnknownCurrencyError{Code: normalized}
	}

	return c, nil
}

// MustCurrency is like LookupCurrency but panics on an unknown code. Use it for
// constants known at compile time.
func MustCurrency(code string) Currency {
	c, err := LookupCurrency(code)
	if err != nil {
		panic(err)
	}

	return c
}

// NewCurrency returns a custom currency, such as loyalty points with no
// decimals: NewCurrency("PTS", 0). An ISO 4217 code keeps its name and numeric
// code with the given precision. Use a Registry (config money.currencies) to
// make custom currencies resolvable by code.
func NewCurrency(code string, minorUnits int) (Currency, error) {
	normalized := normalizeCode(code)
	if normalized == "" || minorUnits < 0 {
		return Currency{}, invalidConfig("currencies", `a map of currency codes to non-negative integer decimal places, e.g. "PTS": 0`)
	}
	if iso, ok := isoCurrencies[normalized]; ok {
		iso.minorUnits = minorUnits

		return iso, nil
	}

	return Currency{code: normalized, name: normalized, minorUnits: minorUnits}, nil
}

// Currencies returns every ISO 4217 currency, sorted by code.
func Currencies() []Currency {
	list := make([]Currency, 0, len(isoCurrencies))
	for _, c := range isoCurrencies {
		list = append(list, c)
	}
	sort.Slice(list, func(i, j int) bool { return list[i].code < list[j].code })

	return list
}

// Code returns the alphabetic code, for example "USD".
func (c Currency) Code() string { return c.code }

// Name returns the ISO 4217 currency name, for example "US Dollar". Custom
// currencies use their code.
func (c Currency) Name() string { return c.name }

// MinorUnits returns the number of decimal places of the currency's minor unit
// (2 for USD, 0 for JPY, 3 for KWD).
func (c Currency) MinorUnits() int { return c.minorUnits }

// NumericCode returns the ISO 4217 numeric code, for example 840 for USD, or 0
// for a custom currency.
func (c Currency) NumericCode() int { return c.numericCode }

// IsZero reports whether c is the zero Currency.
func (c Currency) IsZero() bool { return c.code == "" }

// Equals reports whether c and other are the same currency with the same precision.
func (c Currency) Equals(other Currency) bool { return c == other }

// String returns the alphabetic code.
func (c Currency) String() string { return c.code }

// MarshalText encodes the currency as its alphabetic code.
func (c Currency) MarshalText() ([]byte, error) { return []byte(c.code), nil }

// UnmarshalText decodes a code with the configured registry (ISO 4217 plus
// money.currencies); unknown codes are rejected.
func (c *Currency) UnmarshalText(text []byte) error {
	manager, err := current()
	if err != nil {
		return err
	}
	parsed, err := manager.Registry().Lookup(string(text))
	if err != nil {
		return err
	}
	*c = parsed

	return nil
}

func normalizeCode(code string) string {
	return strings.ToUpper(strings.TrimSpace(code))
}

// Registry is every currency an application knows: ISO 4217 plus the custom
// currencies of config money.currencies, which are checked first so they can
// add codes or override an ISO precision. It is the counterpart of
// laravel-money's CurrencyRegistry and is safe for concurrent use.
type Registry struct {
	custom map[string]Currency
}

// NewRegistry returns a registry with custom currencies given as code =>
// decimal places, for example {"PTS": 0, "MMK": 0}. Codes are case-insensitive.
func NewRegistry(custom map[string]int) (*Registry, error) {
	r := &Registry{custom: make(map[string]Currency, len(custom))}
	for code, minorUnits := range custom {
		c, err := NewCurrency(code, minorUnits)
		if err != nil {
			return nil, err
		}
		r.custom[c.code] = c
	}

	return r, nil
}

// Lookup returns the currency for a code: a custom currency, else ISO 4217.
func (r *Registry) Lookup(code string) (Currency, error) {
	normalized := normalizeCode(code)
	if c, ok := r.custom[normalized]; ok {
		return c, nil
	}

	return LookupCurrency(normalized)
}

// Has reports whether the code is a custom or ISO 4217 currency.
func (r *Registry) Has(code string) bool {
	_, err := r.Lookup(code)

	return err == nil
}

// Custom returns the configured custom currencies as code => decimal places.
func (r *Registry) Custom() map[string]int {
	out := make(map[string]int, len(r.custom))
	for code, c := range r.custom {
		out[code] = c.minorUnits
	}

	return out
}
