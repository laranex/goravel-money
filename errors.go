package money

import (
	"errors"
	"fmt"
)

// Sentinel errors. Every error returned by this package is an
// *InvalidMoneyError that wraps one of them, so callers can use errors.Is for
// the reason and errors.As for the typed error.
var (
	// ErrUnknownCurrency reports a code that is not an ISO 4217 currency.
	ErrUnknownCurrency = errors.New("money: unknown currency")
	// ErrInvalidDecimal reports a string that is not a plain decimal number.
	ErrInvalidDecimal = errors.New("money: invalid decimal amount")
	// ErrOverflow reports an amount that does not fit in int64 minor units.
	ErrOverflow = errors.New("money: amount overflows int64 minor units")
	// ErrCurrencyMismatch reports an operation or column given another currency.
	ErrCurrencyMismatch = errors.New("money: currency mismatch")
	// ErrInvalidStoredAmount reports a database or JSON value that is not an
	// integer amount in minor units.
	ErrInvalidStoredAmount = errors.New("money: invalid stored amount")
)

// InvalidMoneyError is the typed error returned by this package. It is the Go
// counterpart of laravel-money's InvalidMoneyException.
type InvalidMoneyError struct {
	// Err is one of the sentinel errors above.
	Err error
	msg string
}

func newError(reason error, format string, args ...any) *InvalidMoneyError {
	return &InvalidMoneyError{Err: reason, msg: fmt.Sprintf(format, args...)}
}

// Error implements error.
func (e *InvalidMoneyError) Error() string { return "money: " + e.msg }

// Unwrap returns the sentinel reason.
func (e *InvalidMoneyError) Unwrap() error { return e.Err }
