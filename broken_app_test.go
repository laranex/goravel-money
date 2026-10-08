package money

import (
	"encoding/json"
	"errors"
	"testing"

	mocksfoundation "github.com/goravel/framework/mocks/foundation"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/laranex/goravel-money/v4/iso"
)

// TestMisconfiguredApp registers an app whose manager cannot be built (for
// example an unknown money.default_currency): config-dependent calls return
// the error, and calls that cannot fail fall back to the defaults.
func TestMisconfiguredApp(t *testing.T) {
	usd := MustCurrency("USD")
	m := MustParse("10", usd)

	app := mocksfoundation.NewApplication(t)
	app.EXPECT().Make(Binding).Return(nil, errors.New("invalid money config")).Maybe()
	setRegistered(app)
	t.Cleanup(resetRegistered)

	_, err := m.DividedBy(3)
	assert.ErrorContains(t, err, "invalid money config")
	_, err = m.RatioOf(m, 2)
	assert.ErrorContains(t, err, "invalid money config")
	_, err = m.Fields()
	assert.ErrorContains(t, err, "invalid money config")
	_, err = json.Marshal(m)
	assert.ErrorContains(t, err, "invalid money config")
	assert.ErrorContains(t, json.Unmarshal([]byte(`{"amount":"1","currency":"USD"}`), &m), "invalid money config")
	var c Currency
	assert.ErrorContains(t, c.UnmarshalText([]byte("USD")), "invalid money config")
	_, err = NewColumn[iso.USD](m).Value()
	assert.ErrorContains(t, err, "invalid money config")
	var balance AmountColumn
	require.NoError(t, balance.Scan(int64(1)))
	_, _, err = balance.Money("USD")
	assert.ErrorContains(t, err, "invalid money config")

	assert.Equal(t, "3.33", must(t)(m.DividedBy(3, HalfUp)).Decimal(), "an explicit rounding needs no config")
	assert.Equal(t, "10.00", MustParse("9.995", usd, HalfUp).RoundTo(2).Decimal())
	assert.Equal(t, "10.00", MustParse("9.50", usd).RoundTo(0).Decimal(), "RoundTo falls back to HalfUp")
	assert.Equal(t, "USD 10.00", m.Format(), "Format falls back to the plain format")
}

func TestParserEdgeCases(t *testing.T) {
	for _, input := range []string{"1,2a3", "1,23a", "a,123"} {
		_, err := Parse(input, MustCurrency("USD"))
		assert.ErrorIs(t, err, ErrInvalidDecimal, input)
	}
	_, err := MustParse("1", MustCurrency("USD")).DividedBy("x")
	assert.ErrorIs(t, err, ErrInvalidDecimal)
	assert.Panics(t, func() { mustBig("1.5") })
}
