# Changelog

All notable changes to `goravel-money` will be documented in this file.

## v4.0.0 - Unreleased

Initial release, the Goravel counterpart of `laranex/laravel-money`. The version number matches the other Laranex packages, so the module path carries the major version: import `github.com/laranex/goravel-money/v4`.

### Added
- `Currency` with every active ISO 4217 currency (179, the same list as moneyphp/laravel-money), its name, numeric code and minor units: `LookupCurrency`, `MustCurrency`, `Currencies`.
- `Money`, an immutable amount in `int64` minor units of one currency: `New`, `Amount`, `Currency`, `Decimal`, `String`, `Equals`, `Compare`, `Add`, `Subtract` and sign checks; JSON as `{"amount":"1050","currency":"USD"}`.
- `Parse` for plain decimal strings with laravel-money's rules (extra fraction digits round half away from zero; signs other than a leading `-`, separators and exponents are rejected).
- `Manager` with the laravel-money API: `Make` (minor units), `Parse` (decimal string), `Format`, `Currency` and `DefaultCurrency`; an empty currency code means the default currency.
- `ServiceProvider` binding the manager as a singleton, `facades.Money()` accessor, `config/money.go` with `money.default_currency` from `MONEY_CURRENCY` (default `USD`), publishable with `./artisan vendor:publish --package=github.com/laranex/goravel-money/v4` (tags `goravel-money`, `goravel-money-config`), and a `setup` program for `./artisan package:install`.
- `Column[C]`, the counterpart of `MoneyCast`: a `driver.Valuer`/`sql.Scanner`/JSON type that stores integer minor units with the currency fixed by the type parameter (`Column[iso.MMK]`) or the configured default (`Column[money.Default]`); the `iso` package has one marker per ISO 4217 currency.
- `InvalidMoneyError`, the counterpart of `InvalidMoneyException`, wrapping `ErrUnknownCurrency`, `ErrInvalidDecimal`, `ErrOverflow`, `ErrCurrencyMismatch` or `ErrInvalidStoredAmount`.
- Agent skill in `skills/goravel-money`; install it with `npx skills add laranex/goravel-money`.
