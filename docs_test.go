package money

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// order mirrors the introduction example of laravel-money's
// tests/Feature/DocumentationExamplesTest.php.
type order struct {
	ID       uint
	Subtotal Column[Default] `json:"subtotal"`
	Currency string          `json:"currency"`
	Total    AmountColumn    `json:"-"`
}

// TestIntroductionExample is laravel-money's "runs the introduction example".
func TestIntroductionExample(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Locale = "en"
	manager := useConfig(t, cfg)

	subtotal, err := manager.Parse("1,234.50", "USD")
	require.NoError(t, err)
	total, err := subtotal.AddPercent("8.875")
	require.NoError(t, err)
	parts, err := of(t, "100", "").Split(3)
	require.NoError(t, err)
	percentage, err := of(t, "25", "").PercentageOf(of(t, "200", ""), 2)
	require.NoError(t, err)
	yen, err := of(t, "1500", "JPY").Times("1.1")
	require.NoError(t, err)

	assert.Equal(t, "1344.06", total.Decimal())
	assert.Equal(t, []string{"33.34", "33.33", "33.33"}, decimals(parts))
	assert.Equal(t, "12.50", percentage)
	assert.Equal(t, "1650", yen.Decimal())
	assert.True(t, of(t, "1234.50", "USD").Equals(subtotal))

	db := openDB(t)
	require.NoError(t, db.AutoMigrate(&order{}))
	created := order{Subtotal: NewColumn[Default](subtotal)}
	require.NoError(t, created.Total.Set(total, &created.Currency))
	require.NoError(t, db.Create(&created).Error)

	var got order
	require.NoError(t, db.First(&got, created.ID).Error)
	assert.Equal(t, "USD", got.Currency)
	gotTotal, ok, err := got.Total.Money(got.Currency)
	require.NoError(t, err)
	require.True(t, ok)

	data, err := json.Marshal(gotTotal)
	require.NoError(t, err)
	assert.JSONEq(t, `{"amount":"134406","currency":"USD","decimal":"1344.06","formatted":"$1,344.06"}`, string(data))
}

// TestDocumentationExamples is laravel-money's "runs the documentation
// examples", plus every example on the docs website's usage, arithmetic and
// columns pages.
func TestDocumentationExamples(t *testing.T) {
	useConfig(t, laravelConfig())
	price := of(t, "1,234.50", "USD")

	tests := []struct {
		name string
		got  func() (Money, error)
		want string
	}{
		{"addPercent 8.875", func() (Money, error) { return price.AddPercent("8.875") }, "1344.06"},
		{"subtractPercent 15", func() (Money, error) { return price.SubtractPercent(15) }, "1049.32"},
		{"percent 7.5", func() (Money, error) { return of(t, "200", "").Percent("7.5") }, "15.00"},
		{"addPercent 7", func() (Money, error) { return of(t, "200", "").AddPercent(7) }, "214.00"},
		{"subtractPercent 15 of 200", func() (Money, error) { return of(t, "200", "").SubtractPercent(15) }, "170.00"},
		{"zero decimals", func() (Money, error) { return Parse("12.500", MustCurrency("USD")) }, "12.50"},
		{"rounding", func() (Money, error) { return Parse("1.235", MustCurrency("USD"), HalfUp) }, "1.24"},
		{"half even", func() (Money, error) { return Parse("2.5", MustCurrency("JPY"), HalfEven) }, "2"},
		{"times 3", func() (Money, error) { return of(t, "10.00", "").Times(3) }, "30.00"},
		{"times 0.5", func() (Money, error) { return of(t, "0.05", "").Times("0.5") }, "0.03"},
		{"times floor", func() (Money, error) { return of(t, "0.05", "").Times("0.5", Floor) }, "0.02"},
		{"divided 3", func() (Money, error) { return of(t, "20.00", "").DividedBy(3) }, "6.67"},
		{"divided 0.5", func() (Money, error) { return of(t, "10.00", "").DividedBy("0.5") }, "20.00"},
		{"mod", func() (Money, error) { return of(t, "10.00", "").Mod("3") }, "1.00"},
		{"plus", func() (Money, error) { return of(t, "19.99", "").Plus(of(t, "5", ""), "2.50") }, "27.49"},
		{"minus", func() (Money, error) { return of(t, "27.49", "").Minus("7.49") }, "20.00"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m, err := tt.got()
			require.NoError(t, err)
			assert.Equal(t, tt.want, m.Decimal())
		})
	}

	ratio, err := of(t, "50", "").RatioOf(of(t, "200", ""), 4)
	require.NoError(t, err)
	assert.Equal(t, "0.2500", ratio)
	percentage, err := of(t, "2", "").PercentageOf(of(t, "3", ""), 4)
	require.NoError(t, err)
	assert.Equal(t, "66.6667", percentage)

	assert.Equal(t, "12.00", of(t, "12.34", "").RoundTo(0).Decimal())
	assert.Equal(t, "12.00", of(t, "12.50", "").RoundTo(0, HalfEven).Decimal())
	assert.Equal(t, "12.40", of(t, "12.34", "").RoundTo(1, Ceiling).Decimal())
	assert.Equal(t, "20", of(t, "15", "JPY").RoundTo(-1).Decimal())

	kwd, err := of(t, "1", "KWD").Split(6)
	require.NoError(t, err)
	assert.Equal(t, []string{"0.167", "0.167", "0.167", "0.167", "0.166", "0.166"}, decimals(kwd))

	assert.Equal(t, "$1,234.50", price.Format("en"))
	assert.Equal(t, "1.234,50 €", of(t, "1234.5", "EUR").Format("de_DE"))
	assert.Equal(t, "¥1,500", of(t, "1500", "JPY").Format("en"))
	assert.Equal(t, "₹12,34,567.89", of(t, "1234567.89", "INR").Format("en_IN"))
	assert.Equal(t, "USD 1234.50", price.String())
}
