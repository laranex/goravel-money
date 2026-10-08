---
name: goravel-money
description: >
  Handle money in a Goravel application with github.com/laranex/goravel-money/v4: ISO 4217 currencies, integer minor units, decimal parsing and formatting, and money.Column fields for the ORM.
license: MIT
metadata:
  author: Nay Thu Khant
---

# Goravel Money

## When to use

Use this skill when Goravel code creates, parses, formats, stores or serializes monetary amounts. Keep every amount as `money.Money` (integer minor units plus an ISO 4217 currency) from request to database to JSON, never as `float64`.

## Install

```bash
go get github.com/laranex/goravel-money/v4
./artisan package:install github.com/laranex/goravel-money/v4
```

Requires Go 1.25+ and Goravel 1.18+. `package:install` registers `&money.ServiceProvider{}` in `bootstrap/providers.go`, writes `config/money.go` and adds `MONEY_CURRENCY` to `.env.example`. To do it by hand, register the provider and run `./artisan vendor:publish --package=github.com/laranex/goravel-money/v4` (`--tag=goravel-money-config` works only together with `--package`).

## Configure

- `MONEY_CURRENCY` sets `money.default_currency` (default `USD`).
- An unknown code makes `moneyfacades.Money()` panic on first use, like a missing facade.

## Use

Import `money "github.com/laranex/goravel-money/v4"`, `moneyfacades "github.com/laranex/goravel-money/v4/facades"` and, for fixed-currency fields, `"github.com/laranex/goravel-money/v4/iso"`.

### Create, parse and format

`moneyfacades.Money()` returns the `*money.Manager`; an empty currency code means the default currency.

```go
price, err := moneyfacades.Money().Parse(ctx.Request().Input("price"), "") // "10.50" -> 1050 minor units
cost, err := moneyfacades.Money().Make(250000, "MMK")                     // 2500.00 MMK
moneyfacades.Money().Format(price)                                        // "10.50"
price.String()                                                            // "10.50 USD"
```

- `Parse` rounds extra decimals half away from zero (`"10.555"` USD -> 1056) and rejects `"1,000"`, `"+5"` and `"1e3"` with `ErrInvalidDecimal`.
- `DefaultCurrency()` and `Currency(code)` return a `money.Currency` (`Code()`, `Name()`, `MinorUnits()`).
- Minor units come from ISO 4217: USD and MMK have 2, JPY 0, BHD 3.
- Without the container: `money.New(amount, money.MustCurrency("USD"))`, `money.LookupCurrency(code)`, `money.Parse(decimal, currency)`, `m.Decimal()`.

### Arithmetic

`Add`, `Subtract` and `Compare` return `ErrCurrencyMismatch` for different currencies and `ErrOverflow` outside `int64`. `Amount()`, `IsZero()`, `IsPositive()`, `IsNegative()` and `Equals()` read the value.

### Store amounts with the ORM

```go
type Product struct {
	orm.Model
	Price money.Column[money.Default] `json:"price"` // the default currency
	Cost  money.Column[iso.MMK]       `json:"cost"`  // always MMK
}

product := Product{Price: money.NewColumn[money.Default](price), Cost: money.NewColumn[iso.MMK](cost)}
err = facades.Orm().Query().Create(&product) // price = 1050, cost = 250000
```

- Create the columns as integers: `table.BigInteger("price").Nullable()`.
- Saving a `Money` in another currency fails with `ErrCurrencyMismatch`.
- Read with `field.Valid` (false for NULL), `field.Money`, or `field.Ptr()` (nil for NULL).
- JSON is `{"amount":"1050","currency":"USD"}` (minor units as a string) or `null`.

### Handle errors

Every error is a `*money.InvalidMoneyError`; check the reason with `errors.Is(err, money.ErrUnknownCurrency)` (or `ErrInvalidDecimal`, `ErrOverflow`, `ErrCurrencyMismatch`, `ErrInvalidStoredAmount`). Answer invalid user input with HTTP 422 and the error message.

## Test your app

- Code that needs no container can build values directly: `money.New(1050, money.MustCurrency("USD"))` or a `money.NewManager("USD")`.
- Code that calls `moneyfacades.Money()` needs `&money.ServiceProvider{}` registered in the test application, with `MONEY_CURRENCY` set in the test environment.
- Assert on `Amount()` and `Currency().Code()`, or on the JSON shape, never on floats.

## Avoid

- Converting amounts through `float64`, `strconv.ParseFloat` or `math.Round`.
- Storing decimals ("10.50") in the database; store integer minor units through `money.Column`.
- Hard-coding two decimals; JPY has none and BHD has three.
- Mixing currencies in arithmetic or saving a `Money` into a column of another currency.
