---
name: goravel-money
description: >
  Handle money in a Goravel application with github.com/laranex/goravel-money/v4: exact Money values of any size in integer minor units, strict parsing, arithmetic with rounding modes, percentages, allocation, locale-aware formatting, configurable JSON, and ORM column types.
license: MIT
metadata:
  author: Nay Thu Khant
---

# Goravel Money

## When to use

Use this skill when Goravel code creates, parses, calculates, formats, stores or serializes monetary amounts. Keep every amount as `money.Money` from request to database to JSON, never as `float64`.

## Install

```bash
go get github.com/laranex/goravel-money/v4
./artisan package:install github.com/laranex/goravel-money/v4
```

Requires Go 1.25+ and Goravel 1.18+. `package:install` registers `&money.ServiceProvider{}` in `bootstrap/providers.go`, writes `config/money.go` and adds `MONEY_CURRENCY` to `.env.example`. By hand: register the provider and run `./artisan vendor:publish --package=github.com/laranex/goravel-money/v4`.

## Configure

`config/money.go` (`money.*`):

- `default_currency`: `config.Env("MONEY_CURRENCY", "USD")`, used whenever a currency code is empty.
- `rounding`: `"half_up"` (default), `"half_down"`, `"half_even"`, `"half_odd"`, `"half_positive_infinity"`, `"half_negative_infinity"`, `"ceiling"` or `"floor"`.
- `currencies`: custom currencies as code => decimal places, e.g. `"PTS": 0`; they also override ISO precision (`"MMK": 0`).
- `locale`: formatting locale such as `"en"`, `"de_DE"` or `"my_MM"`; empty uses `app.locale`.
- `serialization`: `amount` (`"minor"` or `"decimal"`), `include_decimal`, `include_formatted`.

An unknown default currency or an invalid value makes `moneyfacades.Money()` panic on first use.

## Use

Import `money "github.com/laranex/goravel-money/v4"`, `moneyfacades "github.com/laranex/goravel-money/v4/facades"` and, for fixed-currency columns, `"github.com/laranex/goravel-money/v4/iso"`.

### Create

```go
manager := moneyfacades.Money()                              // *money.Manager
price, err := manager.Parse(ctx.Request().Input("price"), "") // "1,234.50" in the default currency
yen, err := manager.Parse("1500", "JPY")
cents, err := manager.Make(123450, "USD")                     // from int64 minor units
big, err := manager.OfMinor("99999999999999999999", "USD")     // minor units of any size
rounded, err := manager.Parse("1.235", "USD", money.HalfUp)    // 1.24
```

- Parsing is strict: `"12,50"`, `".5"`, `"1e3"`, mixed separators (`"1 234,567"`) and irregular groups (`"1,234,56,789"`) return `money.ErrInvalidDecimal`; grouping needs one separator throughout, in Western groups of three (`"1,234,567"`) or Indian grouping (`"12,34,567"`); `"1.234"` USD returns `money.ErrTooManyDecimals` unless a rounding mode is passed. `"12.500"` is fine.
- Without the container: `money.Parse(amount, money.MustCurrency("USD"))`, `money.MustParse`, `money.New(minor, currency)`, `money.Zero(currency)`.
- Read with `Amount()` (minor units string), `Decimal()` (`"1234.50"`), `Int64()`, `Currency().Code()`, `Precision()`, `IsZero()`, `IsPositive()`, `IsNegative()`.

### Calculate

```go
total, err := price.Plus(shipping, "2.50")           // Money, decimal strings or ints
total, err = total.Minus(discount)
tax, err := price.Times("0.0825")                     // multipliers are strings or ints
each, err := price.DividedBy(3, money.Floor)          // optional rounding per call
withTax, err := price.AddPercent("8.875")
off, err := price.SubtractPercent(15)
pct, err := part.PercentageOf(total, 2)               // "12.50"
parts, err := price.Split(3)                          // 33.34, 33.33, 33.33
shares, err := money.AllocateMap(price, map[string]int{"owner": 70, "agent": 30})
sum, err := money.Sum(prices)                         // also Min, Max, Avg
cash, err := price.RoundTo(0)                         // 12.00
ok, err := price.GreaterThan("10")                    // comparisons return (bool, error)
```

- Mixing currencies returns `money.ErrCurrencyMismatch`; floats return `money.ErrInvalidOperand`; dividing by zero returns `money.ErrDivisionByZero`.
- `PercentageOf`/`RatioOf` scales (0 to `money.MaxScale`, 100) and `RoundTo` decimals (-100 to 100) outside their bounds return `money.ErrInvalidOperand`.
- `Allocate(ratios ...any)` keeps order; `AllocateMap` gives tied leftover cents to keys in ascending order.

### Format and serialize

```go
price.Format()          // configured locale: "$1,234.50"
price.Format("de_DE")   // "1.234,50 €"
price.String()          // "USD 1234.50", never locale-dependent
```

`json.Marshal(price)` gives `{"amount":"123450","currency":"USD","decimal":"1234.50","formatted":"$1,234.50"}` by default; every value is a string. `money.Locales()` lists supported locales; others fall back to their language, then to `"USD 1234.50"`.

### Store amounts with the ORM

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

balance, ok, err := product.Balance.Money(product.Currency) // ok is false for NULL
```

- Migrations: `table.BigInteger("price").Nullable()`, `table.Decimal("fee").Total(20).Places(4).Nullable()`, `table.String("currency", 3).Nullable()`.
- Read `field.Valid` (false for NULL), `field.Money` or `field.Ptr()`.
- Saving Money in another currency than the column's returns `money.ErrCurrencyMismatch`.
- For a custom currency column, declare a marker: `type PTS struct{}` with `func (PTS) CurrencyCode() string { return "PTS" }`, then `money.Column[PTS]`.
- `DecimalAmountColumn` is the DECIMAL version of `AmountColumn`.

### Handle errors

Every error matches `money.ErrMoney` with `errors.Is`; answer user input errors with HTTP 422:

```go
if errors.Is(err, money.ErrMoney) {
	return ctx.Response().Json(http.StatusUnprocessableEntity, http.Json{"message": err.Error()})
}
```

Specific reasons: `ErrInvalidDecimal`, `ErrTooManyDecimals`, `ErrUnknownCurrency`, `ErrCurrencyMismatch`, `ErrInvalidOperand`, `ErrDivisionByZero`, `ErrInvalidAllocation`, `ErrEmptyAggregate`, `ErrOverflow`, `ErrInvalidStoredAmount`, `ErrInvalidConfig`. Typed errors for `errors.As`: `*money.ParseError`, `*money.UnknownCurrencyError`, `*money.CurrencyMismatchError`, `*money.InvalidMoneyError`.

## Test your app

- Code that needs no container builds values directly: `money.MustParse("10.50", money.MustCurrency("USD"))`, or a manager with `money.NewManager(money.DefaultConfig())`.
- Code that calls `moneyfacades.Money()` needs `&money.ServiceProvider{}` registered in the test application.
- Assert on `Amount()`, `Decimal()` or `Equals()`, never on floats.

## Avoid

- `float64`, `strconv.ParseFloat` or `math.Round` for amounts; pass decimal strings.
- Hard-coding two decimals; JPY has none and KWD has three. Use `Precision()`.
- Comparing formatted strings; formatting depends on the locale.
- Storing decimals in BIGINT columns or minor units in DECIMAL columns; pick `Column` or `DecimalColumn`.
- DECIMAL columns on SQLite (stored as floating point); use `Column` there.
