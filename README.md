# Goravel Money

[![Go Reference](https://pkg.go.dev/badge/github.com/laranex/goravel-money/v4.svg)](https://pkg.go.dev/github.com/laranex/goravel-money/v4)
[![Tests](https://github.com/laranex/goravel-money/actions/workflows/tests.yml/badge.svg)](https://github.com/laranex/goravel-money/actions/workflows/tests.yml)
[![License](https://img.shields.io/github/license/laranex/goravel-money.svg?style=flat-square)](LICENSE.md)

Money for Goravel: exact, currency-aware amounts with arithmetic, percentages, allocation, formatting and ORM columns. Built for humans and AI agents.

## Why Goravel Money

Money is easy to get subtly wrong: floats drop cents, "divide by 100" breaks for yen and Kuwaiti dinar, and splitting a bill loses a cent. Goravel Money handles all of that for you, for every currency:

- **Precision comes from the currency.** USD and MMK have 2 decimals, JPY 0, KWD 3, all from ISO 4217. You can add your own currencies, such as loyalty points with 0 decimals.
- **No floats, ever.** Amounts are integers in minor units or decimal strings, and all math is exact. Passing a float returns an error with a hint.
- **Helpers you actually need:** `Plus`, `Minus`, `Times`, `DividedBy`, `Percent`, `AddPercent`, `SubtractPercent`, `PercentageOf`, `Split`, `Allocate`, `RoundTo`, `Sum`, `Avg` and more.
- **ORM column types** for integer and DECIMAL columns, with a fixed currency or a currency column per row.
- **Exact formatting** for the locale (`$1,234.50`, `1.234,50 €`), even for amounts too large for a float.

## Installation

```bash
go get github.com/laranex/goravel-money/v4
./artisan package:install github.com/laranex/goravel-money/v4
```

Requirements: Go 1.25+ and Goravel 1.18+.

## Documentation

The full documentation lives at **[laranex.vercel.app/goravel-money](https://laranex.vercel.app/goravel-money)**:

- [Introduction](https://laranex.vercel.app/goravel-money/introduction): why Goravel Money, requirements and a quick example
- [Installation](https://laranex.vercel.app/goravel-money/installation): installing, setting the default currency and the configuration
- [Usage](https://laranex.vercel.app/goravel-money/usage): creating, comparing, formatting and serializing Money values, and errors
- [Arithmetic](https://laranex.vercel.app/goravel-money/arithmetic): exact math, percentages, splitting, allocation and rounding modes
- [Columns](https://laranex.vercel.app/goravel-money/columns): ORM column types for integer and DECIMAL columns
- [AI Agent Skill](https://laranex.vercel.app/goravel-money/ai-agent-skill): the agent skill for Claude Code, Codex, Cursor and other coding agents

## License

MIT. See [LICENSE.md](LICENSE.md).
