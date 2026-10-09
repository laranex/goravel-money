package money

import (
	"database/sql/driver"
	"encoding/json"
	"math/big"
	"regexp"
	"strconv"
)

// CurrencyCode is a type-level currency used as the type parameter of Column
// and DecimalColumn. The iso sub-package has one marker per ISO 4217 currency
// (iso.MMK, iso.USD, ...); Default uses the configured default currency. For
// a custom currency, declare your own marker:
//
//	type PTS struct{}
//
//	func (PTS) CurrencyCode() string { return "PTS" }
type CurrencyCode interface {
	CurrencyCode() string
}

// Default is the CurrencyCode marker for the configured default currency
// (config money.default_currency).
type Default struct{}

// CurrencyCode returns "", meaning the default currency.
func (Default) CurrencyCode() string { return "" }

var (
	integerPattern = regexp.MustCompile(`^-?\d+$`)
	decimalPattern = regexp.MustCompile(`^-?\d+(\.\d+)?$`)
)

// Column stores Money as integer minor units in a BIGINT column, with the
// currency fixed by the type parameter. It is the counterpart of
// laravel-money's Money::class and AsMoney::of('USD') casts.
//
//	type Product struct {
//		orm.Model
//		Price money.Column[money.Default] // default currency
//		Cost  money.Column[iso.MMK]       // always MMK
//	}
//
// The zero Column is SQL NULL / JSON null. It implements driver.Valuer,
// sql.Scanner and json.Marshaler, so it works with Goravel's ORM (GORM),
// database/sql and JSON responses. Amounts that do not fit in an int64 are
// written as integer strings, for NUMERIC(n, 0) columns.
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
func (c Column[C]) Currency() (Currency, error) { return markerCurrency[C]() }

// Ptr returns the Money, or nil for NULL.
func (c Column[C]) Ptr() *Money { return ptr(c.Money, c.Valid) }

// Value implements driver.Valuer: NULL, or the amount in minor units (int64,
// or an integer string beyond int64). Money in another currency returns a
// *CurrencyMismatchError.
func (c Column[C]) Value() (driver.Value, error) {
	if !c.Valid {
		return nil, nil
	}
	if err := checkMarker[C](c.Money); err != nil {
		return nil, err
	}

	return minorValue(c.Money), nil
}

// Scan implements sql.Scanner. It accepts NULL, integers and integer strings
// (minor units); anything else returns ErrInvalidStoredAmount.
func (c *Column[C]) Scan(src any) error {
	if src == nil {
		*c = Column[C]{}

		return nil
	}
	currency, err := markerCurrency[C]()
	if err != nil {
		return err
	}
	m, err := scanMinor(src, currency)
	if err != nil {
		return err
	}
	*c = Column[C]{Money: m, Valid: true}

	return nil
}

// GormDataType tells GORM's AutoMigrate to use an integer column.
func (Column[C]) GormDataType() string { return "bigint" }

// MarshalJSON encodes the Money (see Money.MarshalJSON), or null.
func (c Column[C]) MarshalJSON() ([]byte, error) { return marshalNullable(c.Money, c.Valid) }

// UnmarshalJSON decodes null or a Money object; the currency must match the
// column's currency.
func (c *Column[C]) UnmarshalJSON(data []byte) error {
	m, valid, err := unmarshalMarker[C](data)
	if err != nil {
		return err
	}
	*c = Column[C]{Money: m, Valid: valid}

	return nil
}

// DecimalColumn stores Money as a decimal string ("12.50") in a DECIMAL
// column, with the currency fixed by the type parameter. It is the
// counterpart of laravel-money's AsMoney::decimal('USD') cast.
//
// Reading is strict: a stored value with more non-zero decimals than the
// currency allows (12.345 for USD) returns a *ParseError; trailing zeros
// (12.3400) are fine. SQLite stores DECIMAL columns as floating point, so a
// float from the database is accepted only when it maps back to an exact
// amount; prefer Column on SQLite.
type DecimalColumn[C CurrencyCode] struct {
	Money Money
	// Valid is false for NULL.
	Valid bool
}

// NewDecimalColumn wraps m for a DecimalColumn field.
func NewDecimalColumn[C CurrencyCode](m Money) DecimalColumn[C] {
	return DecimalColumn[C]{Money: m, Valid: true}
}

// Currency returns the currency the column stores.
func (c DecimalColumn[C]) Currency() (Currency, error) { return markerCurrency[C]() }

// Ptr returns the Money, or nil for NULL.
func (c DecimalColumn[C]) Ptr() *Money { return ptr(c.Money, c.Valid) }

// Value implements driver.Valuer: NULL, or the decimal string such as "12.50".
func (c DecimalColumn[C]) Value() (driver.Value, error) {
	if !c.Valid {
		return nil, nil
	}
	if err := checkMarker[C](c.Money); err != nil {
		return nil, err
	}

	return c.Money.Decimal(), nil
}

// Scan implements sql.Scanner. It accepts NULL, decimal strings, integers and
// floats that map back to an exact amount.
func (c *DecimalColumn[C]) Scan(src any) error {
	if src == nil {
		*c = DecimalColumn[C]{}

		return nil
	}
	currency, err := markerCurrency[C]()
	if err != nil {
		return err
	}
	raw, err := scanDecimalRaw(src)
	if err != nil {
		return err
	}
	m, err := raw.money(currency)
	if err != nil {
		return err
	}
	*c = DecimalColumn[C]{Money: m, Valid: true}

	return nil
}

// GormDataType tells GORM's AutoMigrate to use a DECIMAL column.
func (DecimalColumn[C]) GormDataType() string { return "decimal(38,10)" }

// MarshalJSON encodes the Money (see Money.MarshalJSON), or null.
func (c DecimalColumn[C]) MarshalJSON() ([]byte, error) { return marshalNullable(c.Money, c.Valid) }

// UnmarshalJSON decodes null or a Money object; the currency must match the
// column's currency.
func (c *DecimalColumn[C]) UnmarshalJSON(data []byte) error {
	m, valid, err := unmarshalMarker[C](data)
	if err != nil {
		return err
	}
	*c = DecimalColumn[C]{Money: m, Valid: valid}

	return nil
}

// AmountColumn stores integer minor units whose currency lives in another
// column of the same row, for multi-currency tables. It is the counterpart of
// laravel-money's AsMoney::currencyColumn('currency') cast. Go struct fields
// cannot see each other while GORM scans a row, so the currency is passed
// explicitly:
//
//	type Wallet struct {
//		orm.Model
//		Currency string             // table.String("currency", 3).Nullable()
//		Balance  money.AmountColumn // table.BigInteger("balance").Nullable()
//	}
//
//	err := wallet.Balance.Set(money.MustParse("1500", jpy), &wallet.Currency) // fills Currency with "JPY"
//	balance, err := wallet.Balance.Money(wallet.Currency)                     // 1500 JPY
//
// Several amount columns can share one currency column.
type AmountColumn struct {
	amount string // integer minor units as stored
	// Valid is false for NULL.
	Valid bool
}

// Set stores m and keeps the currency column in step, like laravel-money:
// when *currency is empty it is set to m's currency code; when it holds
// another currency, Set returns a *CurrencyMismatchError and changes nothing.
func (a *AmountColumn) Set(m Money, currency *string) error {
	if err := syncCurrencyColumn(m, currency); err != nil {
		return err
	}
	*a = AmountColumn{amount: m.Amount(), Valid: true}

	return nil
}

// Money returns the stored amount in the given currency (the row's currency
// column); an empty code uses the default currency. NULL returns ok false.
func (a AmountColumn) Money(currency string) (m Money, ok bool, err error) {
	if !a.Valid {
		return Money{}, false, nil
	}
	c, err := resolveCode(currency)
	if err != nil {
		return Money{}, false, err
	}

	return FromBigInt(Money{amount: a.amount}.big(), c), true, nil
}

// Amount returns the stored minor units as an integer string, "" for NULL.
// AmountColumn{Valid: true} built by hand holds zero.
func (a AmountColumn) Amount() string {
	if !a.Valid {
		return ""
	}

	return Money{amount: a.amount}.Amount()
}

// Value implements driver.Valuer: NULL, or the amount in minor units.
func (a AmountColumn) Value() (driver.Value, error) {
	if !a.Valid {
		return nil, nil
	}

	return minorValue(Money{amount: a.amount}), nil
}

// Scan implements sql.Scanner. It accepts NULL, integers and integer strings.
func (a *AmountColumn) Scan(src any) error {
	if src == nil {
		*a = AmountColumn{}

		return nil
	}
	m, err := scanMinor(src, Currency{})
	if err != nil {
		return err
	}
	*a = AmountColumn{amount: m.Amount(), Valid: true}

	return nil
}

// GormDataType tells GORM's AutoMigrate to use an integer column.
func (AmountColumn) GormDataType() string { return "bigint" }

// MarshalJSON encodes the stored minor units as a string, or null. The
// currency is in another field; use Money(currency) to serialize full Money.
func (a AmountColumn) MarshalJSON() ([]byte, error) {
	if !a.Valid {
		return []byte("null"), nil
	}

	return json.Marshal(a.Amount())
}

// DecimalAmountColumn stores a decimal string ("12.50") in a DECIMAL column
// whose currency lives in another column of the same row. It is the
// counterpart of laravel-money's AsMoney::decimal(currencyColumn: 'currency').
// It works like AmountColumn, and reads strictly like DecimalColumn.
type DecimalAmountColumn struct {
	raw rawDecimal
	// Valid is false for NULL.
	Valid bool
}

// Set stores m and keeps the currency column in step (see AmountColumn.Set).
func (a *DecimalAmountColumn) Set(m Money, currency *string) error {
	if err := syncCurrencyColumn(m, currency); err != nil {
		return err
	}
	*a = DecimalAmountColumn{raw: rawDecimal{text: m.Decimal()}, Valid: true}

	return nil
}

// Money returns the stored amount in the given currency (the row's currency
// column); an empty code uses the default currency. NULL returns ok false.
// More non-zero decimals than the currency has return a *ParseError.
func (a DecimalAmountColumn) Money(currency string) (m Money, ok bool, err error) {
	if !a.Valid {
		return Money{}, false, nil
	}
	c, err := resolveCode(currency)
	if err != nil {
		return Money{}, false, err
	}
	m, err = a.raw.money(c)
	if err != nil {
		return Money{}, false, err
	}

	return m, true, nil
}

// Value implements driver.Valuer: NULL, or the decimal string.
func (a DecimalAmountColumn) Value() (driver.Value, error) {
	if !a.Valid {
		return nil, nil
	}
	if a.raw.isFloat {
		return a.raw.float, nil
	}

	return a.raw.text, nil
}

// Scan implements sql.Scanner. It accepts NULL, decimal strings, integers and floats.
func (a *DecimalAmountColumn) Scan(src any) error {
	if src == nil {
		*a = DecimalAmountColumn{}

		return nil
	}
	raw, err := scanDecimalRaw(src)
	if err != nil {
		return err
	}
	*a = DecimalAmountColumn{raw: raw, Valid: true}

	return nil
}

// GormDataType tells GORM's AutoMigrate to use a DECIMAL column.
func (DecimalAmountColumn) GormDataType() string { return "decimal(38,10)" }

// MarshalJSON encodes the stored decimal as a string, or null. The currency
// is in another field; use Money(currency) to serialize full Money.
func (a DecimalAmountColumn) MarshalJSON() ([]byte, error) {
	if !a.Valid {
		return []byte("null"), nil
	}
	if a.raw.isFloat {
		return json.Marshal(strconv.FormatFloat(a.raw.float, 'f', -1, 64))
	}

	return json.Marshal(a.raw.text)
}

// rawDecimal is a stored DECIMAL value before its currency is known.
type rawDecimal struct {
	text    string
	float   float64
	isFloat bool
}

func scanDecimalRaw(src any) (rawDecimal, error) {
	switch v := src.(type) {
	case int64:
		return rawDecimal{text: strconv.FormatInt(v, 10)}, nil
	case int:
		return rawDecimal{text: strconv.Itoa(v)}, nil
	case int32:
		return rawDecimal{text: strconv.FormatInt(int64(v), 10)}, nil
	case float64:
		return rawDecimal{float: v, isFloat: true}, nil
	case []byte:
		return decimalText(string(v), src)
	case string:
		return decimalText(v, src)
	default:
		return rawDecimal{}, invalidStoredDecimal(src)
	}
}

func decimalText(text string, src any) (rawDecimal, error) {
	if !decimalPattern.MatchString(text) {
		return rawDecimal{}, invalidStoredDecimal(src)
	}

	return rawDecimal{text: text}, nil
}

// money reads the stored value strictly in the currency. A float is accepted
// only when it maps back to an exact amount, like laravel-money on SQLite.
func (r rawDecimal) money(currency Currency) (Money, error) {
	if !r.isFloat {
		return Parse(r.text, currency)
	}
	text := strconv.FormatFloat(r.float, 'f', currency.minorUnits, 64)
	back, err := strconv.ParseFloat(text, 64)
	m, parseErr := Parse(text, currency)
	if err != nil || parseErr != nil || back != r.float || new(big.Int).Abs(m.big()).Cmp(maxExactFloat) >= 0 {
		return Money{}, newError(ErrInvalidStoredAmount, "the database returned the float %s, which cannot be read exactly; SQLite stores DECIMAL columns as REAL, so use integer storage (money.Column) instead", strconv.FormatFloat(r.float, 'g', -1, 64))
	}

	return m, nil
}

// maxExactFloat is 2^53, beyond which float64 cannot hold every integer.
var maxExactFloat = new(big.Int).Lsh(big.NewInt(1), 53)

func invalidStoredDecimal(src any) error {
	return newError(ErrInvalidStoredAmount, "the stored value must be a decimal amount, %s given", describe(src))
}

func scanMinor(src any, currency Currency) (Money, error) {
	switch v := src.(type) {
	case int64:
		return New(v, currency), nil
	case int:
		return New(int64(v), currency), nil
	case int32:
		return New(int64(v), currency), nil
	case []byte:
		return minorText(string(v), currency, src)
	case string:
		return minorText(v, currency, src)
	default:
		return Money{}, invalidStoredMinor(src)
	}
}

func minorText(text string, currency Currency, src any) (Money, error) {
	if !integerPattern.MatchString(text) {
		return Money{}, invalidStoredMinor(src)
	}

	return FromBigInt(mustBig(text), currency), nil
}

func invalidStoredMinor(src any) error {
	return newError(ErrInvalidStoredAmount, "the stored value must be an integer amount in minor units, %s given; store minor units in an integer column (e.g. 1050 for 10.50), or use money.DecimalColumn for a DECIMAL column", describe(src))
}

func describe(src any) string {
	switch v := src.(type) {
	case string:
		return strconv.Quote(v)
	case []byte:
		return strconv.Quote(string(v))
	case float64:
		return "float " + strconv.FormatFloat(v, 'g', -1, 64)
	default:
		return typeName(src)
	}
}

// minorValue returns the minor units as int64 when they fit, else as a string.
func minorValue(m Money) driver.Value {
	if n, err := m.Int64(); err == nil {
		return n
	}

	return m.Amount()
}

func ptr(m Money, valid bool) *Money {
	if !valid {
		return nil
	}

	return &m
}

func marshalNullable(m Money, valid bool) ([]byte, error) {
	if !valid {
		return []byte("null"), nil
	}

	return m.MarshalJSON()
}

func markerCurrency[C CurrencyCode]() (Currency, error) {
	var marker C

	return resolveCode(marker.CurrencyCode())
}

// resolveCode resolves a code with the configured registry; "" is the default currency.
func resolveCode(code string) (Currency, error) {
	manager, err := current()
	if err != nil {
		return Currency{}, err
	}

	return manager.Currency(code)
}

func checkMarker[C CurrencyCode](m Money) error {
	currency, err := markerCurrency[C]()
	if err != nil {
		return err
	}
	if !m.currency.Equals(currency) {
		return &CurrencyMismatchError{Expected: currency.code, Given: m}
	}

	return nil
}

func unmarshalMarker[C CurrencyCode](data []byte) (Money, bool, error) {
	if string(data) == "null" {
		return Money{}, false, nil
	}
	var m Money
	if err := json.Unmarshal(data, &m); err != nil {
		return Money{}, false, err
	}
	if err := checkMarker[C](m); err != nil {
		return Money{}, false, err
	}

	return m, true, nil
}

func syncCurrencyColumn(m Money, currency *string) error {
	if currency == nil {
		return newError(ErrInvalidOperand, "Set needs a pointer to the currency column field")
	}
	code := normalizeCode(*currency)
	switch {
	case code == "":
		*currency = m.currency.code
	case code != m.currency.code:
		return &CurrencyMismatchError{Expected: code, Given: m, CurrencyColumn: true}
	}

	return nil
}
