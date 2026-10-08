# Goravel Money

[![Go Reference](https://pkg.go.dev/badge/github.com/laranex/goravel-money/v4.svg)](https://pkg.go.dev/github.com/laranex/goravel-money/v4)
[![Tests](https://img.shields.io/github/actions/workflow/status/laranex/goravel-money/tests.yml?branch=main&label=tests&style=flat-square)](https://github.com/laranex/goravel-money/actions/workflows/tests.yml)
[![License](https://img.shields.io/github/license/laranex/goravel-money.svg?style=flat-square)](LICENSE.md)

Money for [Goravel](https://www.goravel.dev) applications, the Go counterpart of [`laranex/laravel-money`](https://github.com/laranex/laravel-money). It ships ISO 4217 currencies with their minor-unit precision, an immutable `Money` value stored as integer minor units (never floats), decimal parsing and formatting that match laravel-money, a `money.Column` type that stores amounts in integer database columns through Goravel's ORM, and a service provider with a configurable default currency. It is for Goravel developers who need exact amounts in models, requests and JSON responses.

## Documentation

Full documentation lives at **[laranex.vercel.app/goravel-money](https://laranex.vercel.app/goravel-money)**.

## Requirements

- Go 1.25 or higher
- Goravel 1.18 or higher

## Installation

```bash
go get github.com/laranex/goravel-money/v4
./artisan package:install github.com/laranex/goravel-money/v4
```

`package:install` registers `&money.ServiceProvider{}` in `bootstrap/providers.go`, writes `config/money.go` and adds `MONEY_CURRENCY=USD` to `.env.example`. To do it by hand, register the provider and publish the config:

```bash
./artisan vendor:publish --package=github.com/laranex/goravel-money/v4
```

## Usage

```go
import (
	money "github.com/laranex/goravel-money/v4"
	moneyfacades "github.com/laranex/goravel-money/v4/facades"
	"github.com/laranex/goravel-money/v4/iso"
)

price, err := moneyfacades.Money().Parse("10.50", "")   // 1050 minor units in MONEY_CURRENCY
cost, err := moneyfacades.Money().Make(250000, "MMK")   // 2500.00 MMK
moneyfacades.Money().Format(price)                      // "10.50"
moneyfacades.Money().DefaultCurrency().Code()           // "USD"

// Store amounts as integer minor units with Goravel's ORM
type Product struct {
	orm.Model
	Price money.Column[money.Default] `json:"price"` // the default currency
	Cost  money.Column[iso.MMK]       `json:"cost"`  // always MMK
}

product := Product{Price: money.NewColumn[money.Default](price), Cost: money.NewColumn[iso.MMK](cost)}
err = facades.Orm().Query().Create(&product) // price = 1050, cost = 250000
// JSON: {"price":{"amount":"1050","currency":"USD"},"cost":{"amount":"250000","currency":"MMK"}}

var invalid *money.InvalidMoneyError
if errors.As(err, &invalid) && errors.Is(err, money.ErrCurrencyMismatch) { /* ... */ }
```

Create the columns as integers, for example `table.BigInteger("price").Nullable()`. See the [documentation](https://laranex.vercel.app/goravel-money) for parsing rules, arithmetic and errors.

## Built for humans and AI agents

The documentation is written for developers, and the package ships an agent skill so AI coding agents use it the way it's meant to be used.

- Install it with `npx skills add laranex/goravel-money` (Claude Code, Codex, Cursor and others), or copy `skills/goravel-money` into your project's `.claude/skills` or `.agents/skills`.

## Testing

```bash
go test ./...
```

## Changelog

Please see [CHANGELOG](CHANGELOG.md) for more information on what has changed recently.

## Contributing

Please see [CONTRIBUTING](.github/CONTRIBUTING.md) for details.

## Security Vulnerabilities

Please review [our security policy](.github/SECURITY.md) on how to report security vulnerabilities.

## Credits

- [Nay Thu Khant](https://github.com/NayThuKhant)
- [All Contributors](../../contributors)

## License

The MIT License (MIT). Please see [License File](LICENSE.md) for more information.
