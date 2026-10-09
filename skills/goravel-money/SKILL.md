---
name: goravel-money
description: >
  Work with money in a Goravel app using github.com/laranex/goravel-money/v4: the immutable money.Money value type, ORM column types, exact arithmetic, percentages, allocation, rounding and formatting with the correct precision for every currency.
license: MIT
metadata:
  author: Nay Thu Khant
---

# Goravel Money

## When to use

Use this skill when a Goravel app creates, calculates, formats, stores or serializes prices, balances, totals, taxes, discounts or payouts. Keep every amount as `money.Money` from request to database to JSON, never as a `float64`. The currency decides the precision (USD 2, JPY 0, KWD 3, MMK 2), so never hard-code "divide by 100".

## Install

```bash
go get github.com/laranex/goravel-money/v4
./artisan package:install github.com/laranex/goravel-money/v4
```

Requires Go 1.25+ and Goravel 1.18+. `package:install` registers `&money.ServiceProvider{}` in `bootstrap/providers.go`, writes `config/money.go` and adds `MONEY_CURRENCY` to `.env.example`. Formatting needs no extra dependency; without a locale, `Format()` returns `USD 1234.50`.

## Configure

`config/money.go` (`money.*`; by hand, publish it with `./artisan vendor:publish --package=github.com/laranex/goravel-money/v4`):

- `default_currency`: `config.Env("MONEY_CURRENCY", "USD")`, used whenever a currency code is empty.
- `rounding`: `"half_up"` (default), `"half_down"`, `"half_even"`, `"half_odd"`, `"half_positive_infinity"`, `"half_negative_infinity"`, `"ceiling"` or `"floor"`.
- `currencies`: custom currencies as code => decimal places, e.g. `"PTS": 0`; they also override ISO precision (`"MMK": 0`).
- `locale`: formatting locale such as `"en"`, `"de_DE"` or `"my_MM"`; empty uses `app.locale`.
- `serialization`: `amount` (`"minor"` or `"decimal"`), `include_decimal`, `include_formatted`.

An unknown default currency or an invalid value makes `moneyfacades.Money()` panic the first time it is used.

## Use

Import `money "github.com/laranex/goravel-money/v4"`, `moneyfacades "github.com/laranex/goravel-money/v4/facades"` and, for fixed-currency columns, `"github.com/laranex/goravel-money/v4/iso"`.

### Build money

```go
manager := moneyfacades.Money()                         // *money.Manager
price, err := manager.Of(ctx.Request().Input("price"), "") // "1,234.50" in the default currency
yen, err := manager.Of(1500, "JPY")                     // an int is a whole amount
cents, err := manager.OfMinor(123450, "USD")            // from minor units
rounded, err := manager.Of("1.235", "USD", money.HalfUp) // 1.24
zero, err := manager.Zero("KWD")
```

- Parsing is strict: `"12,50"`, `".5"`, `"1e3"`, mixed separators (`"1 234,567"`) and irregular groups (`"1,234,56,789"`) return `money.ErrInvalidDecimal`; grouping needs one separator throughout, in Western groups of three (`"1,234,567"`) or Indian grouping (`"12,34,567"`); `"1.234"` USD returns `money.ErrTooManyDecimals` unless a rounding mode is passed. `"12.500"` is fine.
- Without the container, pass a currency: `money.Of("12.34", money.MustCurrency("USD"))`, `money.MustOf`, `money.OfMinor`, `money.Zero`.
- Read with `Amount()` (minor units string), `Decimal()` (`"1234.50"`), `Currency().Code()`, `Precision()`, `IsZero()`, `IsPositive()`, `IsNegative()`.

### Calculate

```go
total, err := price.Plus(shipping, "2.50")      // Money, decimal strings or ints
total, err = total.Minus(discount)
tax, err := price.Times("0.0825")               // multipliers are strings or ints
each, err := price.DividedBy(3, money.Floor)    // optional rounding per call
withTax, err := price.AddPercent("8.875")
off, err := price.SubtractPercent(15)
pct, err := part.PercentageOf(total, 2)         // "12.50"
parts, err := price.Split(3)                    // 33.34, 33.33, 33.33
shares, err := money.AllocateMap(price, map[string]int{"owner": 70, "agent": 30})
sum, err := money.Sum(prices)                   // also Min, Max, Avg
cash, err := price.RoundTo(0)                   // 12.00
ok, err := price.GreaterThan("10")              // also LessThan, Compare...
```

- Mixing currencies returns `money.ErrCurrencyMismatch`; floats return `money.ErrInvalidOperand`; dividing by zero returns `money.ErrDivisionByZero`.
- `PercentageOf`/`RatioOf` scales (0 to `money.MaxScale`, 100) and `RoundTo` decimals (-100 to 100) outside their bounds return `money.ErrInvalidOperand`.
- `AllocateMap` returns a map; tied leftover cents go to keys in ascending order.

### Format and serialize

```go
price.Format()          // configured locale: "$1,234.50"
price.Format("de_DE")   // "1.234,50 €"
price.String()          // "USD 1234.50", never locale-dependent
```

`json.Marshal(price)` gives `{"amount":"123450","currency":"USD","decimal":"1234.50","formatted":"$1,234.50"}` by default; every value is a string. `money.Locales()` lists the supported locales; others fall back to their language, then to `"USD 1234.50"`.

### Store in the database

```go
type Product struct {
	orm.Model
	Price    money.Column[money.Default]  `json:"price"` // BIGINT minor units, default currency
	Cost     money.Column[iso.MMK]        `json:"cost"`  // BIGINT minor units, always MMK
	Fee      money.DecimalColumn[iso.USD] `json:"fee"`   // DECIMAL "12.50", always USD
	Currency string                       `json:"currency"`
	Balance  money.AmountColumn           `json:"-"`     // BIGINT, currency from Currency
}

product := Product{Price: money.NewColumn[money.Default](price), Fee: money.NewDecimalColumn[iso.USD](fee)}
err = product.Balance.Set(balance, &product.Currency) // fills Currency when empty, mismatch error otherwise
err = facades.Orm().Query().Create(&product)
```

- Migrations: `table.BigInteger("price").Nullable()`, `table.Decimal("fee").Total(20).Places(4).Nullable()`, `table.String("currency", 3).Nullable()`.
- Read `field.Valid` (false for NULL), `field.Money` or `field.Ptr()`; read an `AmountColumn` with `product.Balance.Money(product.Currency)`.
- For a custom currency column, declare a marker: `type PTS struct{}` with `func (PTS) CurrencyCode() string { return "PTS" }`, then `money.Column[PTS]`.
- Saving Money in another currency than the column's returns `money.ErrCurrencyMismatch`.
- `DecimalAmountColumn` is the DECIMAL version of `AmountColumn`.

### Handle errors

Every error matches `money.ErrMoney` with `errors.Is`; answer user input errors with HTTP 422:

```go
price, err := moneyfacades.Money().Of(ctx.Request().Input("price"), "")
if errors.Is(err, money.ErrMoney) {
	return ctx.Response().Json(http.StatusUnprocessableEntity, http.Json{"message": err.Error()})
}
```

Typed errors for `errors.As`: `*money.ParseError` (bad input or too many decimals), `*money.UnknownCurrencyError`, `*money.CurrencyMismatchError`, `*money.InvalidMoneyError` (floats, division by zero, invalid allocations, empty aggregates, invalid stored values, scales out of bounds, invalid config).

## Test your app

- Code that needs no container builds values directly: `money.MustOf("10.50", money.MustCurrency("USD"))`, or a manager with `money.NewManager(money.DefaultConfig())`.
- Code that calls `moneyfacades.Money()` needs `&money.ServiceProvider{}` registered in the test application.
- Assert on `Amount()`, `Decimal()` or `Equals()`, never on floats.

## Avoid

- Floats for amounts (`float64`, `strconv.ParseFloat`, `math.Round`; `Times(1.1)` returns an error); pass decimal strings.
- Hard-coding two decimals; JPY has none and KWD has three. Use `Precision()`.
- Comparing formatted strings; formatting depends on the locale.
- Storing decimals in BIGINT columns or minor units in DECIMAL columns; pick `Column` or `DecimalColumn`.
- DECIMAL columns on SQLite (stored as floating point); use `Column` there.
- Changing a currency's precision in `money.currencies` after amounts are stored in it.
