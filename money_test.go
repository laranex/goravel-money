package money

import (
	"math"
	"math/big"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// The vectors in this file are laravel-money's tests/Feature/MoneyTest.php
// ("construction" and "reading and comparing").

func TestParseUsesEachCurrencysPrecision(t *testing.T) {
	useConfig(t, laravelConfig())

	tests := []struct {
		currency  string
		precision int
		input     string
		minor     string
		decimal   string
	}{
		{"USD", 2, "1234.5", "123450", "1234.50"},
		{"MMK", 2, "1500", "150000", "1500.00"},
		{"JPY", 0, "2500", "2500", "2500"},
		{"KWD", 3, "1.25", "1250", "1.250"},
		{"PTS", 0, "300", "300", "300"},
	}
	for _, tt := range tests {
		t.Run(tt.currency, func(t *testing.T) {
			m := of(t, tt.input, tt.currency)
			assert.Equal(t, tt.precision, m.Precision())
			assert.Equal(t, tt.minor, m.Amount())
			assert.Equal(t, tt.decimal, m.Decimal())
			assert.Equal(t, tt.currency, m.Currency().Code())
		})
	}
}

func TestParseWholeAmounts(t *testing.T) {
	assert.Equal(t, "1200", of(t, "12", "").Amount())
	assert.Equal(t, "12", of(t, "12", "JPY").Amount())
	assert.Equal(t, "-3.000", of(t, "-3", "KWD").Decimal())
}

func TestParseDefaultsToTheConfiguredCurrency(t *testing.T) {
	assert.Equal(t, "USD", of(t, "1", "").Currency().Code())

	useDefaultCurrency(t, "mmk")
	assert.Equal(t, "MMK", of(t, "1", "").Currency().Code())

	manager, err := current()
	require.NoError(t, err)
	zero, err := manager.Zero("")
	require.NoError(t, err)
	assert.Equal(t, "MMK", zero.Currency().Code())
}

func TestParseAcceptsCurrencyCodesInAnyCase(t *testing.T) {
	assert.Equal(t, "EUR", of(t, "1", "eur").Currency().Code())
	assert.Equal(t, "JPY", of(t, "1", " jpy ").Currency().Code())
}

func TestParseStripsGroupingCommasAndSpaces(t *testing.T) {
	tests := []struct{ input, minor string }{
		{"1,234.50", "123450"},
		{"1,234,567.89", "123456789"},
		{"12,34,567.00", "123456700"},
		{"1 234 567.01", "123456701"},
		{"1\u00A0234.50", "123450"},
		{"1\u202F234.50", "123450"},
		{"  99.99  ", "9999"},
		{"+5", "500"},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			assert.Equal(t, tt.minor, of(t, tt.input, "USD").Amount())
		})
	}
}

func TestParseNegativesAndNegativeZero(t *testing.T) {
	assert.Equal(t, "-123456", of(t, "-1,234.56", "").Amount())
	assert.Equal(t, "0", of(t, "-0.00", "").Amount())
	assert.False(t, of(t, "-0", "").IsNegative())
	assert.Equal(t, "7.10", of(t, "007.10", "").Decimal())
}

func TestVeryLargeAmountsKeepEveryDigit(t *testing.T) {
	m := of(t, "123456789012345678901234567890.12", "USD")
	assert.Equal(t, "12345678901234567890123456789012", m.Amount())
	assert.Equal(t, "123456789012345678901234567890.12", m.Decimal())

	plus, err := m.Plus("0.01")
	require.NoError(t, err)
	assert.Equal(t, "123456789012345678901234567890.13", plus.Decimal())

	times, err := m.Times("2")
	require.NoError(t, err)
	assert.Equal(t, "246913578024691357802469135780.24", times.Decimal())

	_, err = m.Int64()
	assert.ErrorIs(t, err, ErrOverflow)
	assert.Equal(t, "12345678901234567890123456789012", m.BigInt().String())
}

func TestParseIsStrictAboutDecimalsTheCurrencyDoesNotHave(t *testing.T) {
	useConfig(t, laravelConfig())

	tests := []struct {
		input, currency, message string
		precision, decimals      int
	}{
		{"1.234", "USD", `money: USD allows 2 decimal places, but "1.234" has 3`, 2, 3},
		{"1.5", "JPY", `money: JPY allows 0 decimal places, but "1.5" has 1`, 0, 1},
		{"0.0001", "KWD", `money: KWD allows 3 decimal places, but "0.0001" has 4`, 3, 4},
		{"10.1", "PTS", "money: PTS allows 0 decimal places", 0, 1},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			manager, err := current()
			require.NoError(t, err)
			_, err = manager.Parse(tt.input, tt.currency)
			require.ErrorIs(t, err, ErrTooManyDecimals)
			assert.ErrorContains(t, err, tt.message)

			var parseErr *ParseError
			require.ErrorAs(t, err, &parseErr)
			assert.Equal(t, tt.currency, parseErr.Currency)
			assert.Equal(t, tt.precision, parseErr.Precision)
			assert.Equal(t, tt.decimals, parseErr.Decimals)
		})
	}

	one, err := NewCurrency("ONE", 1)
	require.NoError(t, err)
	_, err = Parse("1.55", one)
	assert.ErrorContains(t, err, "allows 1 decimal place,", "singular")
}

func TestParseAcceptsExtraDecimalsThatAreZeros(t *testing.T) {
	assert.Equal(t, "1250", of(t, "12.5000", "USD").Amount())
	assert.Equal(t, "100", of(t, "100.0", "JPY").Amount())
}

func TestParseRoundsExtraDecimalsWhenARoundingModeIsGiven(t *testing.T) {
	tests := []struct {
		input, currency string
		rounding        Rounding
		minor           string
	}{
		{"1.235", "USD", HalfUp, "124"},
		{"1.235", "USD", HalfEven, "124"},
		{"1.225", "USD", HalfEven, "122"},
		{"1.239", "USD", Floor, "123"},
		{"-1.231", "USD", Floor, "-124"},
		{"2.5", "JPY", HalfDown, "2"},
		{"2.5", "JPY", HalfUp, "3"},
	}
	for _, tt := range tests {
		t.Run(tt.input+" "+tt.rounding.String(), func(t *testing.T) {
			assert.Equal(t, tt.minor, of(t, tt.input, tt.currency, tt.rounding).Amount())
		})
	}
}

func TestParseRejectsMalformedAmounts(t *testing.T) {
	for _, input := range []string{"", "abc", "1.2.3", "12,50", "1,23", ".5", "5.", "1e3", "1,234.5,0", "--1", "$10", "१२३", "-", "1,,234", ",123", "1,234,", "12 ,345", "１２"} {
		t.Run(input, func(t *testing.T) {
			_, err := Parse(input, MustCurrency("USD"))
			require.ErrorIs(t, err, ErrInvalidDecimal)
			assert.ErrorIs(t, err, ErrMoney)
			assert.EqualError(t, err, `money: cannot parse "`+input+`" as an amount; use digits with a dot as the decimal separator, e.g. "1234.50"; commas or spaces may only group thousands ("1,234.50")`)
		})
	}
}

func TestMustParse(t *testing.T) {
	assert.Equal(t, "1050", MustParse("10.50", MustCurrency("USD")).Amount())
	assert.Equal(t, "124", MustParse("1.235", MustCurrency("USD"), HalfUp).Amount())
	assert.Panics(t, func() { MustParse("1.235", MustCurrency("USD")) })
}

func TestOfMinor(t *testing.T) {
	tests := []struct {
		input, currency, decimal, amount string
	}{
		{"123450", "", "1234.50", "123450"},
		{"-5", "KWD", "-0.005", "-5"},
		{"00042", "JPY", "42", "42"},
		{"+7", "USD", "0.07", "7"},
		{"-0", "USD", "0.00", "0"},
		{"99999999999999999999999", "", "999999999999999999999.99", "99999999999999999999999"},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			m := minor(t, tt.input, tt.currency)
			assert.Equal(t, tt.decimal, m.Decimal())
			assert.Equal(t, tt.amount, m.Amount())
		})
	}

	for _, input := range []string{"10.50", "", "abc", "+-1", "1e3"} {
		_, err := OfMinor(input, MustCurrency("USD"))
		assert.ErrorIs(t, err, ErrInvalidOperand, input)
	}
	_, err := OfMinor("10.50", MustCurrency("USD"))
	assert.EqualError(t, err, `money: minor-unit amounts must be integers such as 1050 or "1050", "10.50" given; use money.Parse for decimal amounts`)
}

func TestNewAndFromBigInt(t *testing.T) {
	usd := MustCurrency("USD")
	assert.Equal(t, "10.50", New(1050, usd).Decimal())
	assert.Equal(t, "-92233720368547758.08", New(math.MinInt64, usd).Decimal())
	assert.Equal(t, "92233720368547758.07", New(math.MaxInt64, usd).Decimal())

	n := big.NewInt(1050)
	m := FromBigInt(n, usd)
	n.SetInt64(1)
	assert.Equal(t, "1050", m.Amount(), "Money does not share the *big.Int")

	got := m.BigInt()
	got.SetInt64(1)
	assert.Equal(t, "1050", m.Amount(), "BigInt returns a copy")

	assert.Equal(t, "0", FromBigInt(nil, usd).Amount())
}

func TestZero(t *testing.T) {
	assert.Equal(t, "0.000", Zero(MustCurrency("KWD")).Decimal())
	assert.True(t, Zero(MustCurrency("USD")).IsZero())
	assert.Equal(t, "0", Money{}.Amount(), "the zero Money is zero")
	assert.True(t, Money{}.IsZero())
	assert.Equal(t, "0", Money{}.Decimal())
}

func TestInt64(t *testing.T) {
	n, err := of(t, "-12.34", "USD").Int64()
	require.NoError(t, err)
	assert.Equal(t, int64(-1234), n)
}

func TestParseRejectsUnknownCurrencies(t *testing.T) {
	manager, err := current()
	require.NoError(t, err)

	_, err = manager.Parse("1", "XYZ")
	require.ErrorIs(t, err, ErrUnknownCurrency)
	assert.ErrorContains(t, err, `money: unknown currency "XYZ"`)

	_, err = manager.OfMinor("1", "PTS")
	assert.ErrorIs(t, err, ErrUnknownCurrency, "PTS is not configured here")
}

func TestSign(t *testing.T) {
	tests := []struct {
		amount                   string
		zero, positive, negative bool
		sign                     int
	}{
		{"0", true, false, false, 0},
		{"0.01", false, true, false, 1},
		{"-0.01", false, false, true, -1},
	}
	for _, tt := range tests {
		t.Run(tt.amount, func(t *testing.T) {
			m := of(t, tt.amount, "")
			assert.Equal(t, tt.zero, m.IsZero())
			assert.Equal(t, tt.positive, m.IsPositive())
			assert.Equal(t, tt.negative, m.IsNegative())
			assert.Equal(t, tt.sign, m.Sign())
		})
	}
}

func TestCompare(t *testing.T) {
	ten := of(t, "10", "")

	assert.True(t, ten.Equals(of(t, "10.00", "")))
	assert.True(t, ten.Equals("10"))
	assert.True(t, ten.Equals(1000-990))
	assert.True(t, ten.Equals(&ten))
	assert.False(t, ten.Equals(of(t, "10", "EUR")))
	assert.False(t, ten.Equals("abc"))
	assert.False(t, ten.Equals(10.0))
	assert.True(t, ten == of(t, "10.00", ""), "Money is comparable")

	tests := []struct {
		name  string
		other any
		want  int
	}{
		{"decimal string below", "9.99", 1},
		{"decimal string equal", "10", 0},
		{"money above", of(t, "11", ""), -1},
		{"integer above", 11, -1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, err := ten.Compare(tt.other)
			require.NoError(t, err)
			assert.Equal(t, tt.want, c)
		})
	}

	check := func(ok bool, err error) bool {
		require.NoError(t, err)

		return ok
	}
	assert.True(t, check(ten.GreaterThan("9.99")))
	assert.True(t, check(ten.GreaterThanOrEqual("10")))
	assert.True(t, check(ten.LessThan(11)))
	assert.True(t, check(ten.LessThanOrEqual("10.00")))
	assert.False(t, check(ten.LessThan("10")))
	assert.False(t, check(ten.GreaterThan("10")))
}

func TestIsSameCurrency(t *testing.T) {
	one := of(t, "1", "")
	assert.True(t, one.IsSameCurrency(of(t, "2", ""), of(t, "3", "")))
	assert.False(t, one.IsSameCurrency(of(t, "2", ""), of(t, "3", "EUR")))
	assert.True(t, one.IsSameCurrency())
}

func TestComparingDifferentCurrenciesFails(t *testing.T) {
	ten, five := of(t, "10", ""), of(t, "5", "EUR")
	for name, call := range map[string]func() (bool, error){
		"GreaterThan":        func() (bool, error) { return ten.GreaterThan(five) },
		"GreaterThanOrEqual": func() (bool, error) { return ten.GreaterThanOrEqual(five) },
		"LessThan":           func() (bool, error) { return ten.LessThan(five) },
		"LessThanOrEqual":    func() (bool, error) { return ten.LessThanOrEqual(five) },
	} {
		t.Run(name, func(t *testing.T) {
			ok, err := call()
			assert.False(t, ok)
			require.ErrorIs(t, err, ErrCurrencyMismatch)
			assert.EqualError(t, err, "money: cannot combine USD 10.00 with EUR 5.00: the amounts are in different currencies; convert one of them first")
		})
	}
}

func TestStringIsPlain(t *testing.T) {
	assert.Equal(t, "USD 1234.50", of(t, "1234.5", "USD").String())
	assert.Equal(t, "KWD -3.000", of(t, "-3", "KWD").String())
	assert.Equal(t, "JPY 15", of(t, "15", "JPY").String())
}
