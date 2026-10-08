package money

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// The vectors in this file are laravel-money's tests/Feature/SerializationTest.php.

func marshal(t *testing.T, value any) string {
	t.Helper()
	data, err := json.Marshal(value)
	require.NoError(t, err)

	return string(data)
}

func TestJSONDefaultShape(t *testing.T) {
	m := of(t, "1234.5", "USD")
	assert.Equal(t, `{"amount":"123450","currency":"USD","decimal":"1234.50","formatted":"USD 1234.50"}`, marshal(t, m))

	cfg := DefaultConfig()
	cfg.Locale = "en"
	useConfig(t, cfg)
	assert.Equal(t, `{"amount":"123450","currency":"USD","decimal":"1234.50","formatted":"$1,234.50"}`, marshal(t, m))

	fields, err := m.Fields()
	require.NoError(t, err)
	assert.Equal(t, []Field{{"amount", "123450"}, {"currency", "USD"}, {"decimal", "1234.50"}, {"formatted", "$1,234.50"}}, fields)
}

func TestJSONDecimalAmount(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Serialization.Amount = AmountDecimal
	useConfig(t, cfg)

	assert.Equal(t, `{"amount":"1.500","currency":"KWD","formatted":"KWD 1.500"}`, marshal(t, of(t, "1.5", "KWD")))

	var m Money
	require.NoError(t, json.Unmarshal([]byte(`{"amount":"1.5","currency":"KWD"}`), &m))
	assert.Equal(t, "1500", m.Amount())
	assert.ErrorIs(t, json.Unmarshal([]byte(`{"amount":"1.5555","currency":"KWD"}`), &m), ErrTooManyDecimals)
}

func TestJSONWithoutDecimalAndFormatted(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Serialization = Serialization{Amount: AmountMinor}
	useConfig(t, cfg)

	assert.Equal(t, `{"amount":"99","currency":"JPY"}`, marshal(t, of(t, "99", "JPY")))
}

func TestJSONRejectsAnInvalidAmountOption(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Serialization.Amount = "cents"
	_, err := NewManager(cfg)
	require.ErrorIs(t, err, ErrInvalidConfig)
	assert.EqualError(t, err, `money: the config value money.serialization.amount must be "minor" or "decimal"`)
}

func TestJSONRoundTrip(t *testing.T) {
	useConfig(t, laravelConfig())

	for _, m := range []Money{of(t, "1234.5", "USD"), of(t, "-0.005", "KWD"), of(t, "300", "PTS"), of(t, "123456789012345678901234567890.12", "USD")} {
		t.Run(m.String(), func(t *testing.T) {
			var decoded Money
			require.NoError(t, json.Unmarshal([]byte(marshal(t, m)), &decoded))
			assert.True(t, m.Equals(decoded))
			assert.Equal(t, m, decoded)
		})
	}
}

func TestJSONDecoding(t *testing.T) {
	tests := []struct {
		name, json, want string
		err              error
	}{
		{"string minor units", `{"amount":"1050","currency":"usd"}`, "USD 10.50", nil},
		{"integer minor units", `{"amount":1050,"currency":"USD"}`, "USD 10.50", nil},
		{"extra keys ignored", `{"amount":"1","currency":"JPY","decimal":"x","formatted":"y"}`, "JPY 1", nil},
		{"decimal in minor mode", `{"amount":"10.50","currency":"USD"}`, "", ErrInvalidStoredAmount},
		{"float number", `{"amount":10.5,"currency":"USD"}`, "", ErrInvalidStoredAmount},
		{"missing amount", `{"currency":"USD"}`, "", ErrInvalidStoredAmount},
		{"null amount", `{"amount":null,"currency":"USD"}`, "", ErrInvalidStoredAmount},
		{"bool amount", `{"amount":true,"currency":"USD"}`, "", ErrInvalidStoredAmount},
		{"bad string", `{"amount":"\x","currency":"USD"}`, "", ErrInvalidStoredAmount},
		{"unknown currency", `{"amount":"1","currency":"XYZ"}`, "", ErrUnknownCurrency},
		{"not an object", `"10.50"`, "", ErrInvalidStoredAmount},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var m Money
			err := m.UnmarshalJSON([]byte(tt.json))
			if tt.err != nil {
				assert.ErrorIs(t, err, tt.err)

				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, m.String())
		})
	}
}

func TestJSONInsideStructs(t *testing.T) {
	type order struct {
		Total Money  `json:"total"`
		Tip   *Money `json:"tip"`
	}
	cfg := DefaultConfig()
	cfg.Serialization.IncludeFormatted = false
	useConfig(t, cfg)

	assert.Equal(t, `{"total":{"amount":"500","currency":"USD","decimal":"5.00"},"tip":null}`, marshal(t, order{Total: of(t, "5", "")}))
}
