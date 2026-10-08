# Changelog

All notable changes to `goravel-money` will be documented in this file.

## v4.0.0 - Unreleased

Initial release, the Goravel counterpart of `laranex/laravel-money` with the same features, rules and numbers. The version number matches the other Laranex packages, so the module path carries the major version: import `github.com/laranex/goravel-money/v4`.

### Added
- `Money`, an immutable amount held as an exact integer number of minor units of **any size** (`math/big` inside, never a float). Amounts are strings in the API and in JSON: `Amount()` is `"123450"`, `Decimal()` is `"1234.50"`; `Int64()` and `BigInt()` convert. `Money` is comparable, so `==` works and it can be a map key.
- Construction: `Parse` (decimal string), `MustParse`, `New` (int64 minor units), `OfMinor` (integer string), `FromBigInt` and `Zero`.
- Strict parsing like laravel-money: commas and spaces only group thousands (`"1,234.50"`, `"12,34,567.00"`), `"12,50"`, `".5"`, `"1e3"` and other shapes are rejected, more decimals than the currency has return `ErrTooManyDecimals` unless a rounding mode is given, and extra decimals that are all zeros are accepted.
- Eight rounding modes, `HalfUp` (default), `HalfDown`, `HalfEven`, `HalfOdd`, `HalfPositiveInfinity`, `HalfNegativeInfinity`, `Ceiling` and `Floor`, configured with `money.rounding` and overridable per call.
- Arithmetic: `Plus` and `Minus` (any number of `Money`, decimal strings or integers), `Times`, `DividedBy`, `Mod`, `Negated`, `Absolute` and `RoundTo`; comparisons `Equals`, `Compare`, `GreaterThan`, `GreaterThanOrEqual`, `LessThan`, `LessThanOrEqual` and `IsSameCurrency`; aggregates `Sum`, `Min`, `Max` and `Avg` over slices. Floats are rejected with `ErrInvalidOperand`.
- Percentages: `Percent`, `AddPercent`, `SubtractPercent`, and `PercentageOf` and `RatioOf` returning decimal strings.
- Allocation that never loses a minor unit, with laravel-money's (and moneyphp's) remainder rules: `Split`, `Allocate` (ordered ratios) and `AllocateMap` (named ratios; ties go to keys in ascending order).
- Exact, locale-aware formatting without ICU or floats: `Format(locale)`, `Manager.Format` and `FormatDecimal`, with CLDR symbols, separators, grouping (including Indian lakh grouping) and native digits for 61 locales, generated from ICU 77.1 and checked against 8,296 ICU outputs. Without a locale, money formats as `USD 1234.50`; `String()` always does.
- Configurable JSON like laravel-money: `{"amount":"123450","currency":"USD","decimal":"1234.50","formatted":"$1,234.50"}` by default, with `money.serialization` (`amount` minor or decimal, `include_decimal`, `include_formatted`); `Fields()` returns the same shape.
- Currencies: every active ISO 4217 currency (179) with name, numeric code and precision (`LookupCurrency`, `MustCurrency`, `Currencies`), custom currencies and ISO precision overrides from `money.currencies` (`NewCurrency`, `Registry`).
- `Manager` (`facades.Money()`): `Parse`, `Make`, `OfMinor`, `Zero`, `Format`, `Currency`, `DefaultCurrency`, `Precision`, `Rounding`, `Locale`, `Serialization` and `Registry`; an empty currency code means the default currency. `NewManager(Config)` and `DefaultConfig()` build one without the container.
- ORM column types for Goravel (GORM), mirroring laravel-money's casts: `Column[C]` (integer minor units, fixed currency), `DecimalColumn[C]` (DECIMAL, fixed currency), `AmountColumn` and `DecimalAmountColumn` (currency read from and written to another column of the row). `C` is `money.Default`, an `iso.*` marker or your own marker for a custom currency.
- Typed errors mirroring laravel-money's exceptions: `*ParseError`, `*UnknownCurrencyError`, `*CurrencyMismatchError` (names both amounts) and `*InvalidMoneyError`, all matching `ErrMoney` and a specific reason with `errors.Is`.
- `ServiceProvider` binding the manager as a singleton, `ConfigFrom`, `config/money.go` (`default_currency`, `rounding`, `currencies`, `locale`, `serialization`) publishable with `./artisan vendor:publish --package=github.com/laranex/goravel-money/v4` (tags `goravel-money`, `goravel-money-config`), and a `setup` program for `./artisan package:install`.
- Agent skill in `skills/goravel-money`; install it with `npx skills add laranex/goravel-money`.
