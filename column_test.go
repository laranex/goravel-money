package money

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"testing"

	mocksfoundation "github.com/goravel/framework/mocks/foundation"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/laranex/goravel-money/v4/iso"
)

// useDefaultCurrency registers a mock application whose manager has the given
// default currency, for Column[Default].
func useDefaultCurrency(t *testing.T, code string) {
	t.Helper()
	manager, err := NewManager(code)
	require.NoError(t, err)
	app := mocksfoundation.NewApplication(t)
	app.EXPECT().Make(Binding).Return(manager, nil).Maybe()
	setRegistered(app)
	t.Cleanup(resetRegistered)
}

func TestColumnCurrency(t *testing.T) {
	resetRegistered()

	c, err := Column[Default]{}.Currency()
	require.NoError(t, err)
	assert.Equal(t, "USD", c.Code(), "outside a Goravel app the default is USD")

	useDefaultCurrency(t, "MMK")
	c, err = Column[Default]{}.Currency()
	require.NoError(t, err)
	assert.Equal(t, "MMK", c.Code())

	c, err = Column[iso.JPY]{}.Currency()
	require.NoError(t, err)
	assert.Equal(t, "JPY", c.Code())

	_, err = Column[unknownMarker]{}.Currency()
	assert.ErrorIs(t, err, ErrUnknownCurrency)
}

type unknownMarker struct{}

func (unknownMarker) CurrencyCode() string { return "XYZ" }

func TestColumnValue(t *testing.T) {
	resetRegistered()
	usd, mmk := MustCurrency("USD"), MustCurrency("MMK")

	tests := []struct {
		name    string
		valuer  driver.Valuer
		want    driver.Value
		wantErr error
	}{
		{"null", Column[Default]{}, nil, nil},
		{"default currency", NewColumn[Default](New(1050, usd)), int64(1050), nil},
		{"fixed currency", NewColumn[iso.MMK](New(250000, mmk)), int64(250000), nil},
		{"negative", NewColumn[iso.MMK](New(-1, mmk)), int64(-1), nil},
		{"mismatch with default", NewColumn[Default](New(1, mmk)), nil, ErrCurrencyMismatch},
		{"mismatch with fixed", NewColumn[iso.MMK](New(1, usd)), nil, ErrCurrencyMismatch},
		{"unknown marker", NewColumn[unknownMarker](New(1, usd)), nil, ErrUnknownCurrency},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.valuer.Value()
			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
				var typed *InvalidMoneyError
				assert.True(t, errors.As(err, &typed))

				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestColumnFollowsTheConfiguredDefault(t *testing.T) {
	useDefaultCurrency(t, "MMK")

	got, err := NewColumn[Default](New(5000, MustCurrency("MMK"))).Value()
	require.NoError(t, err)
	assert.Equal(t, int64(5000), got)

	_, err = NewColumn[Default](New(5000, MustCurrency("USD"))).Value()
	assert.ErrorIs(t, err, ErrCurrencyMismatch)

	var c Column[Default]
	require.NoError(t, c.Scan(int64(5000)))
	assert.Equal(t, "MMK", c.Money.Currency().Code())
}

func TestColumnScan(t *testing.T) {
	resetRegistered()

	tests := []struct {
		name  string
		src   any
		want  int64
		valid bool
	}{
		{"null", nil, 0, false},
		{"int64", int64(1050), 1050, true},
		{"int", 1050, 1050, true},
		{"int32", int32(-5), -5, true},
		{"bytes", []byte("250000"), 250000, true},
		{"string", "42", 42, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := NewColumn[iso.MMK](New(999, MustCurrency("MMK")))
			require.NoError(t, c.Scan(tt.src))
			assert.Equal(t, tt.valid, c.Valid)
			if tt.valid {
				assert.Equal(t, New(tt.want, MustCurrency("MMK")), c.Money)
				assert.Equal(t, &c.Money, c.Ptr())
			} else {
				assert.Nil(t, c.Ptr())
			}
		})
	}
}

func TestColumnScanRejectsNonIntegers(t *testing.T) {
	for name, src := range map[string]any{
		"float":          10.5,
		"decimal string": "10.50",
		"decimal bytes":  []byte("10.50"),
		"bool":           true,
		"empty string":   "",
	} {
		t.Run(name, func(t *testing.T) {
			var c Column[iso.USD]
			assert.ErrorIs(t, c.Scan(src), ErrInvalidStoredAmount)
			assert.False(t, c.Valid)
		})
	}

	var c Column[unknownMarker]
	assert.ErrorIs(t, c.Scan(int64(1)), ErrUnknownCurrency)
}

func TestColumnJSON(t *testing.T) {
	resetRegistered()

	type product struct {
		Price Column[Default] `json:"price"`
		Cost  Column[iso.MMK] `json:"cost"`
	}

	data, err := json.Marshal(product{Price: NewColumn[Default](New(1050, MustCurrency("USD")))})
	require.NoError(t, err)
	assert.JSONEq(t, `{"price":{"amount":"1050","currency":"USD"},"cost":null}`, string(data))

	var decoded product
	require.NoError(t, json.Unmarshal([]byte(`{"price":null,"cost":{"amount":250000,"currency":"MMK"}}`), &decoded))
	assert.False(t, decoded.Price.Valid)
	assert.True(t, decoded.Cost.Valid)
	assert.Equal(t, New(250000, MustCurrency("MMK")), decoded.Cost.Money)

	tests := []struct {
		input   string
		wantErr error
	}{
		{`{"cost":{"amount":"1","currency":"USD"}}`, ErrCurrencyMismatch},
		{`{"cost":{"amount":"1.5","currency":"MMK"}}`, ErrInvalidStoredAmount},
		{`{"cost":{"amount":"1","currency":"XYZ"}}`, ErrUnknownCurrency},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			var p product
			assert.ErrorIs(t, json.Unmarshal([]byte(tt.input), &p), tt.wantErr)
		})
	}

	var bad Column[unknownMarker]
	assert.ErrorIs(t, bad.UnmarshalJSON([]byte(`{"amount":"1","currency":"USD"}`)), ErrUnknownCurrency)
}

func TestColumnGormDataType(t *testing.T) {
	assert.Equal(t, "bigint", Column[Default]{}.GormDataType())
}
