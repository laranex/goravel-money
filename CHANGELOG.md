# Changelog

All notable changes to `goravel-money` will be documented in this file.

## v4.0.0 - Unreleased

Initial release, the Goravel counterpart of `laranex/laravel-money` with the same features, rules and numbers. The version number matches the other Laranex packages, so the module path carries the major version: import `github.com/laranex/goravel-money/v4`.

### Added
- `Money`, an immutable amount held as an exact integer number of minor units of **any size** (`math/big` inside, never a float). Amounts are strings in the API and in JSON: `Amount()` is `"123450"`, `Decimal()` is `"1234.50"`; `Int64()` and `BigInt()` convert. `Money` is comparable, so `==` works and it can be a map key.
- Construction: `Of` (a decimal string, an integer or a `*big.Int`, like laravel-money's `Money::of()`), `MustOf`, `Parse` and `MustParse` (decimal strings only), `New` (int64 minor units), `OfMinor` (an integer, an integer string or a `*big.Int`), `FromBigInt` and `Zero`.
- Strict parsing like laravel-money: ASCII digits only, and commas or spaces only group thousands, consistently: one separator throughout, in Western groups of three (`"1,234,567.50"`) or Indian grouping (`"12,34,567.00"`). Mixed separators (`"1 234,567"`), irregular groups (`"1,234,56,789"`), `"12,50"`, `".5"`, `"1e3"` and other shapes are rejected, more decimals than the currency has return `ErrTooManyDecimals` unless a rounding mode is given, and extra decimals that are all zeros are accepted.
- Eight rounding modes, `HalfUp` (default), `HalfDown`, `HalfEven`, `HalfOdd`, `HalfPositiveInfinity`, `HalfNegativeInfinity`, `Ceiling` and `Floor`, configured with `money.rounding` and overridable per call.
- Arithmetic: `Plus` and `Minus` (any number of `Money`, decimal strings or integers), `Times`, `DividedBy`, `Mod`, `Negated`, `Absolute` and `RoundTo` (returns `(Money, error)`); comparisons `Equals` (returns `(bool, error)`), `Compare`, `GreaterThan`, `GreaterThanOrEqual`, `LessThan`, `LessThanOrEqual` and `IsSameCurrency` (same code and precision); aggregates `Sum`, `Min`, `Max` and `Avg` over slices. Floats are rejected with `ErrInvalidOperand`.
- Percentages: `Percent`, `AddPercent`, `SubtractPercent`, and `PercentageOf` and `RatioOf` returning decimal strings.
- `MaxScale` (100) bounds the `PercentageOf`/`RatioOf` scale (0 to 100) and the `RoundTo` decimals (-100 to 100); larger values return `ErrInvalidOperand`, so no call can force a huge power-of-ten computation.
- Allocation that never loses a minor unit, with laravel-money's (and moneyphp's) remainder rules: `Split`, `Allocate` (ordered ratios) and `AllocateMap` (named ratios; ties go to keys in ascending order).
- Exact, locale-aware formatting without ICU or floats: `Format(locale)`, `Manager.Format` and `FormatDecimal`, with CLDR symbols, separators, grouping (including Indian lakh grouping) and native digits for 61 locales, generated from ICU 77.1 and checked against 8,296 ICU outputs. Without a locale, money formats as `USD 1234.50`; `String()` always does.
- Configurable JSON like laravel-money: `{"amount":"123450","currency":"USD","decimal":"1234.50","formatted":"$1,234.50"}` by default, with `money.serialization` (`amount` minor or decimal, `include_decimal`, `include_formatted`); `Fields()` returns the same shape.
- Currencies: every active ISO 4217 currency (179) with name, numeric code and precision (`LookupCurrency`, `MustCurrency`, `Currencies`), custom currencies and ISO precision overrides from `money.currencies` (`NewCurrency`, `Registry`).
- `Manager` (`facades.Money()`): `Of`, `Parse`, `Make`, `OfMinor`, `Zero`, `Format`, `Currency`, `DefaultCurrency`, `Precision`, `Rounding`, `Locale`, `Serialization` and `Registry`; an empty currency code means the default currency. `NewManager(Config)` and `DefaultConfig()` build one without the container.
- ORM column types for Goravel (GORM), mirroring laravel-money's casts: `Column[C]` (integer minor units, fixed currency), `DecimalColumn[C]` (DECIMAL, fixed currency), `AmountColumn` and `DecimalAmountColumn` (currency read from and written to another column of the row). `C` is `money.Default`, an `iso.*` marker or your own marker for a custom currency.
- Typed errors mirroring laravel-money's exceptions: `*ParseError`, `*UnknownCurrencyError`, `*CurrencyMismatchError` (names both amounts) and `*InvalidMoneyError`, all matching `ErrMoney` and a specific reason with `errors.Is`.
- `ServiceProvider` binding the manager as a singleton, `ConfigFrom`, `config/money.go` (`default_currency`, `rounding`, `currencies`, `locale`, `serialization`) publishable with `./artisan vendor:publish --package=github.com/laranex/goravel-money/v4` (tags `goravel-money`, `goravel-money-config`), and a `setup` program for `./artisan package:install`.
- Agent skill in `skills/goravel-money`; install it with `npx skills add laranex/goravel-money`.

### Changed since the pre-releases
- `RoundTo` now returns `(Money, error)`: it rejects decimals outside -`MaxScale` to `MaxScale`, and returns the config error instead of falling back to `HalfUp` when no rounding is passed and the money config can't be loaded.
- `PercentageOf` and `RatioOf` reject scales above `MaxScale` (100).
- Parsing rejects mixed grouping separators (`"1 234,567"`) and irregular groups (`"1,234,56,789"`, which alpha.1 read as 123456789); normalize such input before parsing.
- Aligned with laravel-money, so the same input gives the same result and the same kind of error in both packages:
  - `Of`, `MustOf` and `Manager.Of` build Money from a decimal string, an integer or a `*big.Int`, like `Money::of()`; floats return `ErrInvalidOperand`. `Parse` and `MustParse` remain for strings.
  - `OfMinor` and `Manager.OfMinor` accept an integer or a `*big.Int` as well as an integer string, like `Money::ofMinor()`.
  - `Equals` returns `(bool, error)`: an operand that is not a valid amount (a float, `"abc"`, too many decimals) returns its error instead of `false`. Different currencies still return `false` with no error.
  - A `Rounding` that is not one of the eight modes, such as `Rounding(42)`, returns `ErrInvalidOperand` wherever a rounding is accepted, including `Parse` and `RoundTo` calls that need no rounding, instead of rounding half up.
