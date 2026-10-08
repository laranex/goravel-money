package money

import (
	"database/sql/driver"
	"encoding/json"
	"strconv"
)

// CurrencyCode is a type-level currency used as the type parameter of Column.
// The iso sub-package has one marker per ISO 4217 currency (iso.MMK, iso.USD,
// ...); Default uses the configured default currency.
type CurrencyCode interface {
	CurrencyCode() string
}

// Default is the CurrencyCode marker for the configured default currency
// (config money.default_currency).
type Default struct{}

// CurrencyCode returns "", meaning the default currency.
func (Default) CurrencyCode() string { return "" }

// Column stores Money in an integer database column as minor units. It is the
// Go counterpart of laravel-money's MoneyCast: the currency is fixed per field
// by the type parameter, so the column only holds the amount.
//
//	type Product struct {
//		orm.Model
//		Price money.Column[money.Default] // default currency
//		Cost  money.Column[iso.MMK]       // always MMK
//	}
//
// The zero Column is SQL NULL / JSON null. It implements driver.Valuer,
// sql.Scanner and json.Marshaler, so it works with Goravel's ORM (GORM),
// database/sql and JSON responses.
type Column[C CurrencyCode] struct {
	Money Money
	// Valid is false for NULL.
	Valid bool
}

// NewColumn wraps m for a Column field. The currency is checked when the value
// is written (Value) and must match the column's currency.
func NewColumn[C CurrencyCode](m Money) Column[C] {
	return Column[C]{Money: m, Valid: true}
}

// Currency returns the currency the column stores.
func (c Column[C]) Currency() (Currency, error) {
	var marker C
	code := marker.CurrencyCode()
	if code == "" {
		return defaultCurrency()
	}

	return LookupCurrency(code)
}

// Ptr returns the Money, or nil for NULL.
func (c Column[C]) Ptr() *Money {
	if !c.Valid {
		return nil
	}
	m := c.Money

	return &m
}

// Value implements driver.Valuer: NULL, or the amount in minor units as int64.
// Money in another currency returns an error wrapping ErrCurrencyMismatch.
func (c Column[C]) Value() (driver.Value, error) {
	if !c.Valid {
		return nil, nil
	}
	currency, err := c.Currency()
	if err != nil {
		return nil, err
	}
	if !c.Money.currency.Equals(currency) {
		return nil, newError(ErrCurrencyMismatch, "the column stores %s amounts, %s given", currency.code, c.Money.currency.code)
	}

	return c.Money.amount, nil
}

// Scan implements sql.Scanner. It accepts NULL, integers and integer strings
// (minor units); anything else returns an error wrapping ErrInvalidStoredAmount.
func (c *Column[C]) Scan(src any) error {
	if src == nil {
		*c = Column[C]{}

		return nil
	}

	var amount int64
	switch v := src.(type) {
	case int64:
		amount = v
	case int:
		amount = int64(v)
	case int32:
		amount = int64(v)
	case []byte:
		parsed, err := strconv.ParseInt(string(v), 10, 64)
		if err != nil {
			return newError(ErrInvalidStoredAmount, "the stored value must be an integer amount in minor units, %q given", string(v))
		}
		amount = parsed
	case string:
		parsed, err := strconv.ParseInt(v, 10, 64)
		if err != nil {
			return newError(ErrInvalidStoredAmount, "the stored value must be an integer amount in minor units, %q given", v)
		}
		amount = parsed
	default:
		return newError(ErrInvalidStoredAmount, "the stored value must be an integer amount in minor units, %T given", src)
	}

	currency, err := c.Currency()
	if err != nil {
		return err
	}
	*c = Column[C]{Money: New(amount, currency), Valid: true}

	return nil
}

// GormDataType tells GORM's AutoMigrate to use an integer column.
func (Column[C]) GormDataType() string { return "bigint" }

// MarshalJSON encodes {"amount":"1050","currency":"USD"}, or null.
func (c Column[C]) MarshalJSON() ([]byte, error) {
	if !c.Valid {
		return []byte("null"), nil
	}

	return c.Money.MarshalJSON()
}

// UnmarshalJSON decodes null or {"amount":...,"currency":...}; the currency
// must match the column's currency.
func (c *Column[C]) UnmarshalJSON(data []byte) error {
	if string(data) == "null" {
		*c = Column[C]{}

		return nil
	}
	var m Money
	if err := json.Unmarshal(data, &m); err != nil {
		return err
	}
	currency, err := c.Currency()
	if err != nil {
		return err
	}
	if !m.currency.Equals(currency) {
		return newError(ErrCurrencyMismatch, "the column stores %s amounts, %s given", currency.code, m.currency.code)
	}
	*c = NewColumn[C](m)

	return nil
}
