// Package money is the Goravel version of laranex/laravel-money: ISO 4217
// currencies, Money as integer minor units, decimal parsing and formatting, a
// Column type for Goravel's ORM, and a service provider with a configurable
// default currency.
//
//	m, err := facades.Money().Parse("10.50", "")  // 1050 in the default currency
//	facades.Money().Format(m)                        // "10.50"
//	usd := money.MustCurrency("USD")
//	money.New(1050, usd).String()                    // "10.50 USD"
package money
