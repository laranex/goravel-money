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
		// The ISO 4217 currency code used by the money facade and by
		// money.Column[money.Default] fields whenever no explicit currency is
		// given. A field may fix its own currency: money.Column[iso.MMK].
		"default_currency": config.Env("MONEY_CURRENCY", "USD"),
	})
}
`)
}
