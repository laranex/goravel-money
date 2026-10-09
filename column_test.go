package money

import (
	"database/sql/driver"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/laranex/goravel-money/v4/iso"
)

// PTS is a custom currency marker, as an application would declare it.
type PTS struct{}

func (PTS) CurrencyCode() string { return "PTS" }

type unknownMarker struct{}

func (unknownMarker) CurrencyCode() string { return "XYZ" }

func TestColumnCurrency(t *testing.T) {
	resetRegistered()

	c, err := Column[Default]{}.Currency()
	require.NoError(t, err)
	assert.Equal(t, "USD", c.Code(), "outside a Goravel app the default is USD")

	cfg := laravelConfig()
	cfg.DefaultCurrency = "MMK"
	useConfig(t, cfg)

	c, err = Column[Default]{}.Currency()
	require.NoError(t, err)
	assert.Equal(t, "MMK", c.Code())

	c, err = DecimalColumn[iso.JPY]{}.Currency()
	require.NoError(t, err)
	assert.Equal(t, "JPY", c.Code())

	c, err = Column[PTS]{}.Currency()
	require.NoError(t, err)
	assert.Equal(t, 0, c.MinorUnits(), "custom currencies come from the registry")

	_, err = Column[unknownMarker]{}.Currency()
	assert.ErrorIs(t, err, ErrUnknownCurrency)
}

func TestColumnValue(t *testing.T) {
	resetRegistered()
	usd, mmk := MustCurrency("USD"), MustCurrency("MMK")
	huge := MustParse("123456789012345678901234.56", usd)

	tests := []struct {
		name    string
		valuer  driver.Valuer
		want    driver.Value
		wantErr error
	}{
		{"null", Column[Default]{}, nil, nil},
		{"default currency", NewColumn[Default](New(1050, usd)), int64(1050), nil},
		{"fixed currency", NewColumn[iso.MMK](New(-250000, mmk)), int64(-250000), nil},
		{"beyond int64", NewColumn[iso.USD](huge), "12345678901234567890123456", nil},
		{"mismatch", NewColumn[iso.MMK](New(1, usd)), nil, ErrCurrencyMismatch},
		{"unknown marker", NewColumn[unknownMarker](New(1, usd)), nil, ErrUnknownCurrency},
		{"decimal null", DecimalColumn[iso.USD]{}, nil, nil},
		{"decimal", NewDecimalColumn[iso.USD](New(1234, usd)), "12.34", nil},
		{"decimal mismatch", NewDecimalColumn[iso.MMK](New(1, usd)), nil, ErrCurrencyMismatch},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.valuer.Value()
			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)

				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}

	_, err := NewColumn[iso.MMK](New(1, usd)).Value()
	assert.EqualError(t, err, "money: the column stores MMK amounts, but USD 0.01 was given; convert it to MMK first")
}

func TestColumnScan(t *testing.T) {
	resetRegistered()

	tests := []struct {
		name    string
		src     any
		want    string
		valid   bool
		wantErr string
	}{
		{"null", nil, "", false, ""},
		{"int64", int64(1050), "USD 10.50", true, ""},
		{"int", 7, "USD 0.07", true, ""},
		{"int32", int32(-5), "USD -0.05", true, ""},
		{"bytes", []byte("1050"), "USD 10.50", true, ""},
		{"string", "-1050", "USD -10.50", true, ""},
		{"huge string", "99999999999999999999", "USD 999999999999999999.99", true, ""},
		{"decimal string", "10.50", "", false, `the stored value must be an integer amount in minor units, "10.50" given`},
		{"exponent", "1e3", "", false, "minor units"},
		{"padded", " 10", "", false, "minor units"},
		{"empty", "", "", false, "minor units"},
		{"float", 1.5, "", false, "float 1.5 given"},
		{"bool", true, "", false, "bool given"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			column := NewColumn[Default](New(1, MustCurrency("EUR")))
			err := column.Scan(tt.src)
			if tt.wantErr != "" {
				require.ErrorIs(t, err, ErrInvalidStoredAmount)
				assert.ErrorContains(t, err, tt.wantErr)

				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.valid, column.Valid)
			if tt.valid {
				assert.Equal(t, tt.want, column.Money.String())
			}
		})
	}

	var unknown Column[unknownMarker]
	assert.ErrorIs(t, unknown.Scan(int64(1)), ErrUnknownCurrency)
}

func TestDecimalColumnScan(t *testing.T) {
	resetRegistered()

	tests := []struct {
		name    string
		src     any
		want    string
		wantErr error
		message string
	}{
		{"null", nil, "", nil, ""},
		{"string", "12.34", "USD 12.34", nil, ""},
		{"trailing zeros", "12.3400", "USD 12.34", nil, ""},
		{"bytes", []byte("-0.50"), "USD -0.50", nil, ""},
		{"int64", int64(12), "USD 12.00", nil, ""},
		{"int", 3, "USD 3.00", nil, ""},
		{"int32", int32(-3), "USD -3.00", nil, ""},
		{"exact float", 12.5, "USD 12.50", nil, ""},
		{"too many decimals", "12.345", "", ErrTooManyDecimals, ""},
		{"text", "abc", "", ErrInvalidStoredAmount, `the stored value must be a decimal amount, "abc" given`},
		{"inexact float", 12.345, "", ErrInvalidStoredAmount, "the database returned the float 12.345, which cannot be read exactly"},
		{"huge float", 1.0e20, "", ErrInvalidStoredAmount, "cannot be read exactly"},
		{"bool", false, "", ErrInvalidStoredAmount, "bool given"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var column DecimalColumn[iso.USD]
			err := column.Scan(tt.src)
			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
				assert.ErrorContains(t, err, tt.message)

				return
			}
			require.NoError(t, err)
			if tt.src == nil {
				assert.False(t, column.Valid)
				assert.Nil(t, column.Ptr())

				return
			}
			assert.Equal(t, tt.want, column.Money.String())
			assert.Equal(t, tt.want, column.Ptr().String())
		})
	}

	var unknown DecimalColumn[unknownMarker]
	assert.ErrorIs(t, unknown.Scan("1"), ErrUnknownCurrency)
}

func TestColumnPtr(t *testing.T) {
	assert.Nil(t, Column[Default]{}.Ptr())
	column := NewColumn[Default](New(5, MustCurrency("USD")))
	p := column.Ptr()
	require.NotNil(t, p)
	assert.Equal(t, "5", p.Amount())
}

func TestColumnJSON(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Serialization.IncludeFormatted = false
	useConfig(t, cfg)

	type row struct {
		Price Column[Default]        `json:"price"`
		Cost  Column[iso.MMK]        `json:"cost"`
		Fee   DecimalColumn[iso.KWD] `json:"fee"`
		Tip   DecimalColumn[iso.KWD] `json:"tip"`
	}
	in := row{
		Price: NewColumn[Default](New(1050, MustCurrency("USD"))),
		Fee:   NewDecimalColumn[iso.KWD](New(1500, MustCurrency("KWD"))),
	}
	data, err := json.Marshal(in)
	require.NoError(t, err)
	assert.JSONEq(t, `{"price":{"amount":"1050","currency":"USD","decimal":"10.50"},"cost":null,"fee":{"amount":"1500","currency":"KWD","decimal":"1.500"},"tip":null}`, string(data))

	var out row
	require.NoError(t, json.Unmarshal(data, &out))
	assert.Equal(t, in, out)

	var cost Column[iso.MMK]
	assert.ErrorIs(t, json.Unmarshal([]byte(`{"amount":"1","currency":"USD"}`), &cost), ErrCurrencyMismatch)
	assert.ErrorIs(t, json.Unmarshal([]byte(`{"amount":"x","currency":"MMK"}`), &cost), ErrInvalidStoredAmount)
	var fee DecimalColumn[iso.KWD]
	assert.ErrorIs(t, json.Unmarshal([]byte(`{"amount":"1","currency":"USD"}`), &fee), ErrCurrencyMismatch)
}

func TestAmountColumnSet(t *testing.T) {
	usd, jpy := MustCurrency("USD"), MustCurrency("JPY")

	var balance AmountColumn
	currency := ""
	require.NoError(t, balance.Set(MustParse("1500", jpy), &currency))
	assert.Equal(t, "JPY", currency, "an empty currency column is filled")
	assert.Equal(t, "1500", balance.Amount())

	currency = "jpy"
	require.NoError(t, balance.Set(MustParse("20", jpy), &currency), "the code is case-insensitive")

	currency = "MMK"
	err := balance.Set(MustParse("1", usd), &currency)
	require.ErrorIs(t, err, ErrCurrencyMismatch)
	assert.EqualError(t, err, "money: the currency column holds MMK, but USD 1.00 was given; convert the amount, or change the currency column first")
	assert.Equal(t, "20", balance.Amount(), "a failed Set changes nothing")
	assert.Equal(t, "MMK", currency)

	assert.ErrorIs(t, balance.Set(MustParse("1", usd), nil), ErrInvalidOperand)

	var total DecimalAmountColumn
	currency = ""
	require.NoError(t, total.Set(MustParse("3.125", MustCurrency("KWD")), &currency))
	assert.Equal(t, "KWD", currency)
	value, err := total.Value()
	require.NoError(t, err)
	assert.Equal(t, "3.125", value)
	assert.ErrorIs(t, total.Set(MustParse("1", usd), &currency), ErrCurrencyMismatch)
}

func TestAmountColumnMoney(t *testing.T) {
	resetRegistered()

	var balance AmountColumn
	m, ok, err := balance.Money("USD")
	require.NoError(t, err)
	assert.False(t, ok, "NULL")
	assert.True(t, m.IsZero())

	require.NoError(t, balance.Scan(int64(1050)))
	m, ok, err = balance.Money("usd")
	require.NoError(t, err)
	assert.True(t, ok)
	assert.Equal(t, "USD 10.50", m.String())

	m, _, err = balance.Money("KWD")
	require.NoError(t, err)
	assert.Equal(t, "KWD 1.050", m.String(), "the row's currency decides the precision")

	useDefaultCurrency(t, "EUR")
	m, _, err = balance.Money("")
	require.NoError(t, err)
	assert.Equal(t, "EUR 10.50", m.String(), "an empty currency column uses the default currency")

	_, _, err = balance.Money("XYZ")
	assert.ErrorIs(t, err, ErrUnknownCurrency)

	// A valid column built by hand, without Set or Scan, holds zero.
	handmade := AmountColumn{Valid: true}
	m, ok, err = handmade.Money("USD")
	require.NoError(t, err)
	assert.True(t, ok)
	assert.Equal(t, "USD 0.00", m.String())
	assert.Equal(t, "0", handmade.Amount())
	data, err := json.Marshal(handmade)
	require.NoError(t, err)
	assert.Equal(t, `"0"`, string(data))
	value, err := handmade.Value()
	require.NoError(t, err)
	assert.Equal(t, int64(0), value)
}

func TestAmountColumnScanAndValue(t *testing.T) {
	var balance AmountColumn
	for _, src := range []any{int64(5), 5, int32(5), []byte("5"), "5"} {
		require.NoError(t, balance.Scan(src))
		assert.Equal(t, "5", balance.Amount())
	}
	require.NoError(t, balance.Scan("123456789012345678901234567890"))
	value, err := balance.Value()
	require.NoError(t, err)
	assert.Equal(t, "123456789012345678901234567890", value)

	require.NoError(t, balance.Scan(int64(-7)))
	value, err = balance.Value()
	require.NoError(t, err)
	assert.Equal(t, int64(-7), value)
	assert.Equal(t, "bigint", balance.GormDataType())

	data, err := json.Marshal(balance)
	require.NoError(t, err)
	assert.Equal(t, `"-7"`, string(data))

	assert.ErrorIs(t, balance.Scan("1.5"), ErrInvalidStoredAmount)
	require.NoError(t, balance.Scan(nil))
	assert.False(t, balance.Valid)
	value, err = balance.Value()
	require.NoError(t, err)
	assert.Nil(t, value)
	data, err = json.Marshal(balance)
	require.NoError(t, err)
	assert.Equal(t, "null", string(data))
}

func TestDecimalAmountColumn(t *testing.T) {
	resetRegistered()

	var total DecimalAmountColumn
	_, ok, err := total.Money("USD")
	require.NoError(t, err)
	assert.False(t, ok)

	require.NoError(t, total.Scan("7.25"))
	m, ok, err := total.Money("EUR")
	require.NoError(t, err)
	assert.True(t, ok)
	assert.Equal(t, "EUR 7.25", m.String())

	_, _, err = total.Money("JPY")
	assert.ErrorIs(t, err, ErrTooManyDecimals, "read strictly in the row's currency")
	_, _, err = total.Money("XYZ")
	assert.ErrorIs(t, err, ErrUnknownCurrency)

	require.NoError(t, total.Scan(12.5))
	m, _, err = total.Money("USD")
	require.NoError(t, err)
	assert.Equal(t, "USD 12.50", m.String())
	value, err := total.Value()
	require.NoError(t, err)
	assert.Equal(t, 12.5, value)
	data, err := json.Marshal(total)
	require.NoError(t, err)
	assert.Equal(t, `"12.5"`, string(data))

	require.NoError(t, total.Scan(12.345))
	_, _, err = total.Money("USD")
	assert.ErrorIs(t, err, ErrInvalidStoredAmount)

	require.NoError(t, total.Scan([]byte("-1")))
	data, err = json.Marshal(total)
	require.NoError(t, err)
	assert.Equal(t, `"-1"`, string(data))
	assert.ErrorIs(t, total.Scan("x"), ErrInvalidStoredAmount)
	assert.Equal(t, "decimal(38,10)", total.GormDataType())

	require.NoError(t, total.Scan(nil))
	value, err = total.Value()
	require.NoError(t, err)
	assert.Nil(t, value)
	data, err = json.Marshal(total)
	require.NoError(t, err)
	assert.Equal(t, "null", string(data))
}

func TestGormDataTypes(t *testing.T) {
	assert.Equal(t, "bigint", Column[Default]{}.GormDataType())
	assert.Equal(t, "decimal(38,10)", DecimalColumn[Default]{}.GormDataType())
}
