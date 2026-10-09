package money

import (
	"errors"
	"fmt"
	"reflect"
)

// Sentinel errors. Every error returned by this package matches ErrMoney and
// one reason below with errors.Is, and is one of four typed errors (the Go
// counterparts of laravel-money's exceptions) for errors.As:
//
//	*ParseError            MoneyParseException          ErrInvalidDecimal, ErrTooManyDecimals
//	*UnknownCurrencyError  UnknownCurrencyException     ErrUnknownCurrency
//	*CurrencyMismatchError CurrencyMismatchException    ErrCurrencyMismatch
//	*InvalidMoneyError     InvalidMoneyException        every other reason
var (
	// ErrMoney is matched by every error from this package (laravel-money's MoneyException).
	ErrMoney = errors.New("money")

	// ErrInvalidDecimal reports a string that is not a decimal amount.
	ErrInvalidDecimal = errors.New("money: invalid decimal amount")
	// ErrTooManyDecimals reports an amount with more decimals than its currency has.
	ErrTooManyDecimals = errors.New("money: too many decimal places")
	// ErrUnknownCurrency reports a code that is neither ISO 4217 nor a configured custom currency.
	ErrUnknownCurrency = errors.New("money: unknown currency")
	// ErrCurrencyMismatch reports amounts in different currencies combined, compared or stored together.
	ErrCurrencyMismatch = errors.New("money: currency mismatch")

	// ErrInvalidOperand reports a float or another value that is not money or a number.
	ErrInvalidOperand = errors.New("money: invalid operand")
	// ErrDivisionByZero reports a division, remainder or ratio by zero.
	ErrDivisionByZero = errors.New("money: division by zero")
	// ErrInvalidAllocation reports invalid split parts or allocation ratios.
	ErrInvalidAllocation = errors.New("money: invalid allocation")
	// ErrEmptyAggregate reports Sum, Min, Max or Avg of an empty list.
	ErrEmptyAggregate = errors.New("money: empty aggregate")
	// ErrOverflow reports an amount that does not fit in an int64 (Int64 only;
	// Money itself has no size limit).
	ErrOverflow = errors.New("money: amount overflows int64")
	// ErrInvalidStoredAmount reports a database or JSON value that is not a valid amount.
	ErrInvalidStoredAmount = errors.New("money: invalid stored amount")
	// ErrInvalidConfig reports an invalid money.* config value.
	ErrInvalidConfig = errors.New("money: invalid config")
)

// ParseError reports an amount that cannot be parsed, or that has more
// decimals than its currency allows and no rounding mode was given.
type ParseError struct {
	// Input is the amount as given.
	Input string
	// Currency is the currency code, empty for multipliers and ratios.
	Currency string
	// Precision is the number of decimals the currency allows.
	Precision int
	// Decimals is the number of decimals Input has (ErrTooManyDecimals only).
	Decimals int
	// Err is ErrInvalidDecimal or ErrTooManyDecimals.
	Err error
}

// Error implements error.
func (e *ParseError) Error() string {
	if errors.Is(e.Err, ErrTooManyDecimals) {
		plural := "s"
		if e.Precision == 1 {
			plural = ""
		}

		return fmt.Sprintf("money: %s allows %d decimal place%s, but %q has %d; pass a rounding mode to round it, e.g. money.Of(%q, currency, money.HalfUp)",
			e.Currency, e.Precision, plural, e.Input, e.Decimals, e.Input)
	}

	return fmt.Sprintf(`money: cannot parse %q as an amount; use digits with a dot as the decimal separator, e.g. "1234.50"; commas or spaces may only group thousands, one separator used consistently ("1,234,567.50" or "12,34,567.50")`, e.Input)
}

// Unwrap returns the reason.
func (e *ParseError) Unwrap() error { return e.Err }

// Is reports whether target is ErrMoney.
func (e *ParseError) Is(target error) bool { return target == ErrMoney }

// UnknownCurrencyError reports a currency code that is neither ISO 4217 nor a
// configured custom currency.
type UnknownCurrencyError struct {
	// Code is the currency code, upper-cased.
	Code string
	// Default is true when the code is the configured default currency.
	Default bool
}

// Error implements error.
func (e *UnknownCurrencyError) Error() string {
	if e.Default {
		return fmt.Sprintf("money: the default currency %q (config money.default_currency / env MONEY_CURRENCY) is not a known currency; use an ISO 4217 code or register it under money.currencies", e.Code)
	}
	example := e.Code
	if example == "" {
		example = "PTS"
	}

	return fmt.Sprintf(`money: unknown currency %q; use an ISO 4217 code (USD, EUR, JPY, MMK...) or register a custom currency in config/money.go under "currencies", e.g. "%s": 2`, e.Code, example)
}

// Unwrap returns ErrUnknownCurrency.
func (e *UnknownCurrencyError) Unwrap() error { return ErrUnknownCurrency }

// Is reports whether target is ErrMoney.
func (e *UnknownCurrencyError) Is(target error) bool { return target == ErrMoney }

// CurrencyMismatchError reports amounts in different currencies that were
// combined or compared, or money stored in a column of another currency. The
// message names both amounts (or the column's currency and the amount).
type CurrencyMismatchError struct {
	// Left is the receiver of an arithmetic or comparison; nil for columns.
	Left *Money
	// Expected is the currency code that was required.
	Expected string
	// Given is the amount in the other currency.
	Given Money
	// CurrencyColumn is true when the expected currency came from a currency column.
	CurrencyColumn bool
}

// Error implements error.
func (e *CurrencyMismatchError) Error() string {
	switch {
	case e.Left != nil:
		return fmt.Sprintf("money: cannot combine %s with %s: the amounts are in different currencies; convert one of them first", e.Left, e.Given)
	case e.CurrencyColumn:
		return fmt.Sprintf("money: the currency column holds %s, but %s was given; convert the amount, or change the currency column first", e.Expected, e.Given)
	default:
		return fmt.Sprintf("money: the column stores %s amounts, but %s was given; convert it to %s first", e.Expected, e.Given, e.Expected)
	}
}

// Unwrap returns ErrCurrencyMismatch.
func (e *CurrencyMismatchError) Unwrap() error { return ErrCurrencyMismatch }

// Is reports whether target is ErrMoney.
func (e *CurrencyMismatchError) Is(target error) bool { return target == ErrMoney }

// InvalidMoneyError reports every other misuse: floats, division by zero,
// invalid allocations, empty aggregates, overflow, invalid stored values and
// invalid config. It is the counterpart of laravel-money's InvalidMoneyException.
type InvalidMoneyError struct {
	// Err is the reason, one of the sentinel errors.
	Err error
	msg string
}

func newError(reason error, format string, args ...any) *InvalidMoneyError {
	return &InvalidMoneyError{Err: reason, msg: fmt.Sprintf(format, args...)}
}

// Error implements error.
func (e *InvalidMoneyError) Error() string { return "money: " + e.msg }

// Unwrap returns the reason.
func (e *InvalidMoneyError) Unwrap() error { return e.Err }

// Is reports whether target is ErrMoney.
func (e *InvalidMoneyError) Is(target error) bool { return target == ErrMoney }

func divisionByZero() error {
	return newError(ErrDivisionByZero, "cannot divide money by zero")
}

func invalidConfig(key, expected string) error {
	return newError(ErrInvalidConfig, "the config value money.%s must be %s", key, expected)
}

func floatOperand(value any) error {
	text := fmt.Sprint(value)

	return newError(ErrInvalidOperand, "floats are not accepted for money (%s given) because they cannot hold decimal amounts exactly; pass a string such as %q, or an integer", text, text)
}

func unsupportedOperand(value any) error {
	return newError(ErrInvalidOperand, "expected money.Money, a decimal string or an integer, %s given", typeName(value))
}

func typeName(value any) string {
	if value == nil {
		return "nil"
	}

	return reflect.TypeOf(value).String()
}
