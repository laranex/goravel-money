package money

import (
	"bytes"
	"encoding/json"
	"strings"
)

// Field is one key of the serialized shape of Money.
type Field struct {
	Key, Value string
}

// Fields returns the serialized shape of m, configured by money.serialization,
// in a fixed order: amount, currency, then decimal and formatted when enabled.
// With the defaults:
//
//	amount: "123450", currency: "USD", decimal: "1234.50", formatted: "$1,234.50"
func (m Money) Fields() ([]Field, error) {
	manager, err := current()
	if err != nil {
		return nil, err
	}

	return manager.Fields(m), nil
}

// Fields returns the serialized shape of money with this manager's settings.
func (m *Manager) Fields(money Money) []Field {
	options := m.serialization
	amount := money.Amount()
	if options.Amount == AmountDecimal {
		amount = money.Decimal()
	}
	fields := []Field{{"amount", amount}, {"currency", money.currency.code}}
	if options.IncludeDecimal && options.Amount != AmountDecimal {
		fields = append(fields, Field{"decimal", money.Decimal()})
	}
	if options.IncludeFormatted {
		fields = append(fields, Field{"formatted", m.Format(money)})
	}

	return fields
}

// MarshalJSON encodes the configured shape with every value as a string, by
// default {"amount":"123450","currency":"USD","decimal":"1234.50","formatted":"$1,234.50"}.
func (m Money) MarshalJSON() ([]byte, error) {
	fields, err := m.Fields()
	if err != nil {
		return nil, err
	}
	var b bytes.Buffer
	b.WriteByte('{')
	for i, field := range fields {
		if i > 0 {
			b.WriteByte(',')
		}
		key, _ := json.Marshal(field.Key)
		value, _ := json.Marshal(field.Value)
		b.Write(key)
		b.WriteByte(':')
		b.Write(value)
	}
	b.WriteByte('}')

	return b.Bytes(), nil
}

// UnmarshalJSON decodes {"amount": ..., "currency": ...}. The amount is read
// as configured by money.serialization.amount: integer minor units (a string
// or a JSON integer) by default, or a decimal string. Other keys, such as
// "decimal" and "formatted", are ignored. The currency is resolved with the
// configured registry.
func (m *Money) UnmarshalJSON(data []byte) error {
	var raw struct {
		Amount   json.RawMessage `json:"amount"`
		Currency string          `json:"currency"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return newError(ErrInvalidStoredAmount, "cannot decode money JSON: %v", err)
	}
	manager, err := current()
	if err != nil {
		return err
	}
	currency, err := manager.Registry().Lookup(raw.Currency)
	if err != nil {
		return err
	}
	amount, err := jsonAmount(raw.Amount)
	if err != nil {
		return err
	}

	var parsed Money
	if manager.Serialization().Amount == AmountDecimal {
		parsed, err = Parse(amount, currency)
	} else {
		parsed, err = OfMinor(amount, currency)
		if err != nil {
			err = newError(ErrInvalidStoredAmount, "the JSON amount must be an integer in minor units, %q given", amount)
		}
	}
	if err != nil {
		return err
	}
	*m = parsed

	return nil
}

func jsonAmount(raw json.RawMessage) (string, error) {
	raw = bytes.TrimSpace(raw)
	if len(raw) > 0 && raw[0] == '"' {
		var s string
		if err := json.Unmarshal(raw, &s); err != nil {
			return "", newError(ErrInvalidStoredAmount, "invalid JSON amount %s", raw)
		}

		return s, nil
	}
	text := string(raw)
	if text == "" || text == "null" || strings.ContainsAny(text, "{[tf") {
		return "", newError(ErrInvalidStoredAmount, "the JSON amount must be a string or a number, %s given", orEmpty(text))
	}

	return text, nil
}

func orEmpty(text string) string {
	if text == "" {
		return "nothing"
	}

	return text
}
