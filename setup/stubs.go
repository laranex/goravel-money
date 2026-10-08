package main

import "strings"

// configStub returns config/money.go for an application whose config package
// is configPackage and whose facades live at facadesImport.
func configStub(configPackage, facadesImport string) string {
	return strings.NewReplacer("DummyPackage", configPackage, "DummyFacades", facadesImport).Replace(`package DummyPackage

import (
	"DummyFacades"
)

func init() {
	config := facades.Config()
	config.Add("money", map[string]any{
		// Default Currency
		//
		// The currency code used by facades.Money() and by
		// money.Column[money.Default] fields whenever no currency is given. It
		// must be an ISO 4217 code or one of the custom currencies below.
		"default_currency": config.Env("MONEY_CURRENCY", "USD"),

		// Default Rounding
		//
		// How results that fall between two minor units are rounded by Times,
		// DividedBy, Percent, Avg, RoundTo and friends. Every method also takes
		// a money.Rounding to override it per call.
		//
		// Supported: "half_up", "half_down", "half_even", "half_odd",
		//            "half_positive_infinity", "half_negative_infinity",
		//            "ceiling", "floor"
		"rounding": "half_up",

		// Custom Currencies
		//
		// Extra currency codes and their number of decimal places, e.g. loyalty
		// points: "PTS": 0. Entries here take precedence over ISO 4217, so they
		// can also override the precision of an ISO currency.
		"currencies": map[string]any{
			// "PTS": 0,
		},

		// Formatting Locale
		//
		// The locale used by Format and the "formatted" JSON key, e.g. "en",
		// "de_DE" or "my_MM". Empty uses the application locale (app.locale).
		"locale": "",

		// Serialization
		//
		// The JSON shape of money.Money. Amounts are always strings.
		//
		// amount:            "minor" ("123450") or "decimal" ("1234.50")
		// include_decimal:   add a "decimal" key when amount is "minor"
		// include_formatted: add a "formatted" key ("$1,234.50")
		"serialization": map[string]any{
			"amount":            "minor",
			"include_decimal":   true,
			"include_formatted": true,
		},
	})
}
`)
}
