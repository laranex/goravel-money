// Package money is the Goravel version of laranex/laravel-money: an immutable
// Money value held as an exact integer number of minor units of any size,
// strict decimal parsing, exact arithmetic with eight rounding modes,
// percentages, allocation without losing a cent, locale-aware formatting
// built from CLDR data without floats, configurable JSON, ORM column types
// for Goravel (GORM), and a service provider with config/money.go.
//
//	usd := money.MustCurrency("USD")
//	price, err := money.Parse("1,234.50", usd)  // 1234.50 USD
//	total, err := price.AddPercent("8.875")     // 1344.06 USD
//	parts, err := total.Split(3)                // 448.02, 448.02, 448.02
//	total.Format("en")                          // "$1,344.06"
//
// Inside a Goravel application, facades.Money() returns the configured
// *Manager: facades.Money().Parse("10.50", "") uses money.default_currency.
package money
