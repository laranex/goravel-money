---
name: goravel-money
description: >
  Handle money in a Goravel application with github.com/laranex/goravel-money/v4: ISO 4217 currencies, integer minor units, decimal parsing and formatting, and money.Column fields for the ORM.
license: MIT
metadata:
  author: Nay Thu Khant
---

# Goravel Money

Use this skill when Goravel code creates, parses, formats, stores or serializes monetary amounts.

## Primary Goal

- keep every amount as `money.Money` (integer minor units plus an ISO 4217 currency) from request to database to JSON, never as `float64`

## Workflow

### 1. Install

- `go get github.com/laranex/goravel-money/v4` (Go 1.25+, Goravel 1.18+), then `./artisan package:install github.com/laranex/goravel-money/v4`: it registers `&money.ServiceProvider{}` in `bootstrap/providers.go`, writes `config/money.go` and adds `MONEY_CURRENCY` to `.env.example`
- by hand: add the provider and run `./artisan vendor:publish --package=github.com/laranex/goravel-money/v4` (`--tag=goravel-money-config` only together with `--package`)
- the default currency is `money.default_currency`, read from `MONEY_CURRENCY` (default `USD`); an unknown code makes `facades.Money()` panic at first use

### 2. Create, parse and format

- import `money "github.com/laranex/goravel-money/v4"`, `moneyfacades "github.com/laranex/goravel-money/v4/facades"` and, for fixed-currency fields, `"github.com/laranex/goravel-money/v4/iso"`
- `moneyfacades.Money()` returns the `*money.Manager`; an empty currency code means the default currency:
  - `Make(1050, "")` minor units -> 10.50 in the default currency; `Make(250000, "MMK")` -> 2500.00 MMK
  - `Parse("10.50", "")` for user input; extra decimals round half away from zero (`"10.555"` USD -> 1056); `"1,000"`, `"+5"`, `"1e3"` return `ErrInvalidDecimal`
  - `Format(m)` -> `"10.50"`; `m.String()` -> `"10.50 USD"`; `DefaultCurrency()`, `Currency(code)`
- without the container: `money.New(amount, money.MustCurrency("USD"))`, `money.Parse(decimal, currency)`, `m.Decimal()`
- minor units come from ISO 4217: USD/MMK 2, JPY 0, BHD 3; read them with `currency.MinorUnits()`
- `Add`, `Subtract` and `Compare` return `ErrCurrencyMismatch` for different currencies and `ErrOverflow` outside `int64`

### 3. Store amounts with the ORM

- model fields: `money.Column[money.Default]` (configured default currency) or `money.Column[iso.MMK]` (fixed currency); the column holds only the integer amount
- migration: `table.BigInteger("price").Nullable()`
- write with `money.NewColumn[iso.MMK](m)`; a `Money` in another currency fails on save with `ErrCurrencyMismatch`
- read with `field.Valid` (false for NULL), `field.Money`, or `field.Ptr()` (nil for NULL)
- JSON is `{"amount":"1050","currency":"USD"}` (amount as a string of minor units) or `null`

### 4. Handle errors

- every error is `*money.InvalidMoneyError`; check the reason with `errors.Is(err, money.ErrUnknownCurrency | ErrInvalidDecimal | ErrOverflow | ErrCurrencyMismatch | ErrInvalidStoredAmount)`
- answer invalid user input with HTTP 422 and the error message

## Examples

- request input: `price, err := moneyfacades.Money().Parse(ctx.Request().Input("price"), "")`, then `product.Price = money.NewColumn[money.Default](price)` and `facades.Orm().Query().Create(&product)`
- display: `moneyfacades.Money().Format(product.Price.Money)` -> `"10.50"`

## Anti-patterns

- do not convert amounts through `float64`, `strconv.ParseFloat` or `math.Round`
- do not store decimals ("10.50") in the database; store the integer minor units through `money.Column`
- do not hard-code two decimals; JPY has none and BHD has three
- do not mix currencies in arithmetic or save a `Money` into a column of another currency
