package money

import (
	"sort"
	"strings"
)

// Currency is an ISO 4217 currency with its minor-unit precision.
//
// The zero Currency is not a valid currency; obtain one with LookupCurrency,
// MustCurrency or a Manager.
type Currency struct {
	code        string
	name        string
	minorUnits  int
	numericCode int
}

// LookupCurrency returns the ISO 4217 currency for an alphabetic code such as
// "USD" or "mmk". Unknown codes return an *InvalidMoneyError wrapping
// ErrUnknownCurrency.
func LookupCurrency(code string) (Currency, error) {
	c, ok := isoCurrencies[strings.ToUpper(strings.TrimSpace(code))]
	if !ok {
		return Currency{}, newError(ErrUnknownCurrency, "unknown ISO 4217 currency %q", code)
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

// Currencies returns every supported ISO 4217 currency, sorted by code.
func Currencies() []Currency {
	list := make([]Currency, 0, len(isoCurrencies))
	for _, c := range isoCurrencies {
		list = append(list, c)
	}
	sort.Slice(list, func(i, j int) bool { return list[i].code < list[j].code })

	return list
}

// Code returns the ISO 4217 alphabetic code, for example "USD".
func (c Currency) Code() string { return c.code }

// Name returns the ISO 4217 currency name, for example "US Dollar".
func (c Currency) Name() string { return c.name }

// MinorUnits returns the number of decimal places of the currency's minor unit
// (2 for USD, 0 for JPY, 3 for BHD).
func (c Currency) MinorUnits() int { return c.minorUnits }

// NumericCode returns the ISO 4217 numeric code, for example 840 for USD.
func (c Currency) NumericCode() int { return c.numericCode }

// IsZero reports whether c is the zero Currency.
func (c Currency) IsZero() bool { return c.code == "" }

// Equals reports whether c and other are the same currency.
func (c Currency) Equals(other Currency) bool { return c.code == other.code }

// String returns the alphabetic code.
func (c Currency) String() string { return c.code }

// MarshalText encodes the currency as its alphabetic code.
func (c Currency) MarshalText() ([]byte, error) { return []byte(c.code), nil }

// UnmarshalText decodes an alphabetic code; unknown codes are rejected.
func (c *Currency) UnmarshalText(text []byte) error {
	parsed, err := LookupCurrency(string(text))
	if err != nil {
		return err
	}
	*c = parsed

	return nil
}
