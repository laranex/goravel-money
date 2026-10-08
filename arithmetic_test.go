package money

import (
	"encoding/json"
	"math/big"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// The vectors in this file are laravel-money's tests/Feature/MoneyTest.php
// ("arithmetic", "percentages and ratios", "allocation" and "rounding").

// must returns a checker for (Money, error) results: must(t)(m.Plus("1")).
func must(t *testing.T) func(Money, error) Money {
	t.Helper()

	return func(m Money, err error) Money {
		t.Helper()
		require.NoError(t, err)

		return m
	}
}

func TestPlusAndMinus(t *testing.T) {
	price := of(t, "19.99", "")

	assert.Equal(t, "25.00", must(t)(price.Plus(of(t, "5.01", ""))).Decimal())
	assert.Equal(t, "21.10", must(t)(price.Plus("0.01", 1, of(t, "0.10", ""))).Decimal())
	assert.Equal(t, "-0.01", must(t)(price.Minus("20")).Decimal())
	assert.True(t, must(t)(price.Minus(of(t, "9.99", ""), "10")).IsZero())
	assert.Equal(t, "19.99", price.Decimal(), "operands never change the receiver")
	assert.Equal(t, "19.99", must(t)(price.Plus()).Decimal())
	assert.Equal(t, "20.00", must(t)(price.Plus(json.Number("0.01"), big.NewInt(0), uint8(0))).Decimal())
}

func TestOperandsAreStrict(t *testing.T) {
	_, err := of(t, "1", "JPY").Plus("0.5")
	require.ErrorIs(t, err, ErrTooManyDecimals)
	assert.ErrorContains(t, err, "JPY allows 0 decimal places")
}

func TestOperandsRejectFloatsAndOtherTypes(t *testing.T) {
	one := of(t, "1", "")
	tests := []struct {
		name    string
		call    func() error
		message string
	}{
		{"Plus float", func() error { _, err := one.Plus(0.1); return err }, `money: floats are not accepted for money (0.1 given) because they cannot hold decimal amounts exactly; pass a string such as "0.1", or an integer`},
		{"Times float", func() error { _, err := one.Times(1.5); return err }, "floats are not accepted"},
		{"Percent float", func() error { _, err := one.Percent(float32(7.5)); return err }, "floats are not accepted"},
		{"Allocate float", func() error { _, err := one.Allocate(0.5, 0.5); return err }, "floats are not accepted"},
		{"Compare float", func() error { _, err := one.Compare(12.5); return err }, "(12.5 given)"},
		{"Plus bool", func() error { _, err := one.Plus(true); return err }, "expected money.Money, a decimal string or an integer, bool given"},
		{"Minus nil", func() error { _, err := one.Minus(nil); return err }, "nil given"},
		{"Times money", func() error { _, err := one.Times(one); return err }, "money.Money given"},
		{"Times nil big", func() error { _, err := one.Times((*big.Int)(nil)); return err }, "*big.Int given"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.call()
			require.ErrorIs(t, err, ErrInvalidOperand)
			assert.ErrorContains(t, err, tt.message)
		})
	}
}

func TestPlusAndMinusRejectDifferentCurrencies(t *testing.T) {
	_, err := of(t, "1", "").Plus(of(t, "1", "MMK"))
	require.ErrorIs(t, err, ErrCurrencyMismatch)
	assert.ErrorContains(t, err, "cannot combine USD 1.00 with MMK 1.00")

	var mismatch *CurrencyMismatchError
	require.ErrorAs(t, err, &mismatch)
	assert.Equal(t, "USD 1.00", mismatch.Left.String())
	assert.Equal(t, "MMK 1.00", mismatch.Given.String())
	assert.Equal(t, "USD", mismatch.Expected)

	_, err = of(t, "1", "").Minus(of(t, "1", "JPY"))
	assert.ErrorIs(t, err, ErrCurrencyMismatch)
}

func TestTimes(t *testing.T) {
	tests := []struct {
		amount, currency string
		multiplier       any
		rounding         []Rounding
		want             string
	}{
		{"10.00", "", 3, nil, "30.00"},
		{"10.00", "", "1.5", nil, "15.00"},
		{"0.05", "", "0.5", nil, "0.03"},
		{"0.05", "", "0.5", []Rounding{HalfDown}, "0.02"},
		{"0.05", "", "0.5", []Rounding{Floor}, "0.02"},
		{"-0.05", "", "0.5", nil, "-0.03"},
		{"100", "JPY", "-0.333", nil, "-33"},
		{"1500", "JPY", "1.1", nil, "1650"},
	}
	for _, tt := range tests {
		t.Run(tt.amount, func(t *testing.T) {
			assert.Equal(t, tt.want, must(t)(of(t, tt.amount, tt.currency).Times(tt.multiplier, tt.rounding...)).Decimal())
		})
	}
}

func TestDividedBy(t *testing.T) {
	tests := []struct {
		amount, currency string
		divisor          any
		rounding         []Rounding
		want             string
	}{
		{"10.00", "", 3, nil, "3.33"},
		{"20.00", "", 3, nil, "6.67"},
		{"20.00", "", 3, []Rounding{Floor}, "6.66"},
		{"10.00", "", "0.5", nil, "20.00"},
		{"10.00", "", "-4", nil, "-2.50"},
		{"1", "KWD", 8, nil, "0.125"},
	}
	for _, tt := range tests {
		t.Run(tt.amount, func(t *testing.T) {
			assert.Equal(t, tt.want, must(t)(of(t, tt.amount, tt.currency).DividedBy(tt.divisor, tt.rounding...)).Decimal())
		})
	}

	for _, divisor := range []any{0, "0.00"} {
		_, err := of(t, "1", "").DividedBy(divisor)
		require.ErrorIs(t, err, ErrDivisionByZero)
		assert.EqualError(t, err, "money: cannot divide money by zero")
	}
}

func TestMod(t *testing.T) {
	assert.Equal(t, "1.00", must(t)(of(t, "10.00", "").Mod("3")).Decimal())
	assert.Equal(t, "0.10", must(t)(of(t, "10.00", "").Mod(of(t, "0.30", ""))).Decimal())
	assert.Equal(t, "-1.00", must(t)(of(t, "-10.00", "").Mod("3")).Decimal(), "the remainder has the sign of the dividend")

	_, err := of(t, "10", "").Mod("0")
	assert.ErrorIs(t, err, ErrDivisionByZero)
	_, err = of(t, "10", "").Mod(of(t, "1", "EUR"))
	assert.ErrorIs(t, err, ErrCurrencyMismatch)
}

func TestNegatedAndAbsolute(t *testing.T) {
	assert.Equal(t, "-5.00", of(t, "5", "").Negated().Decimal())
	assert.Equal(t, "5.00", of(t, "-5", "").Negated().Decimal())
	assert.Equal(t, "0", of(t, "0", "").Negated().Amount())
	assert.Equal(t, "5.00", of(t, "-5", "").Absolute().Decimal())
	assert.Equal(t, "5.00", of(t, "5", "").Absolute().Decimal())
}

func TestAggregates(t *testing.T) {
	list := []Money{of(t, "10", ""), of(t, "2.50", ""), of(t, "7.51", "")}

	assert.Equal(t, "20.01", must(t)(Sum(list)).Decimal())
	assert.Equal(t, "1.00", must(t)(Sum([]Money{of(t, "1", "")})).Decimal())
	assert.Equal(t, "2.50", must(t)(Min(list)).Decimal())
	assert.Equal(t, "10.00", must(t)(Max(list)).Decimal())
	assert.Equal(t, "6.67", must(t)(Avg(list)).Decimal())
	assert.Equal(t, "6.67", must(t)(Avg(list, Floor)).Decimal())

	cents := []Money{of(t, "0.01", ""), of(t, "0.02", "")}
	assert.Equal(t, "0.01", must(t)(Avg(cents, Floor)).Decimal())
	assert.Equal(t, "0.02", must(t)(Avg(cents)).Decimal())
}

func TestAggregatesRejectEmptyAndMixedLists(t *testing.T) {
	for name, call := range map[string]func([]Money) (Money, error){
		"Sum": Sum, "Min": Min, "Max": Max, "Avg": func(l []Money) (Money, error) { return Avg(l) },
	} {
		t.Run(name, func(t *testing.T) {
			_, err := call(nil)
			require.ErrorIs(t, err, ErrEmptyAggregate)
			assert.EqualError(t, err, "money: money."+name+" needs at least one Money")

			_, err = call([]Money{of(t, "1", ""), of(t, "1", "EUR")})
			assert.ErrorIs(t, err, ErrCurrencyMismatch)
		})
	}
}

func TestPercent(t *testing.T) {
	tests := []struct {
		amount, currency string
		percent          any
		rounding         []Rounding
		want             string
	}{
		{"200", "", 15, nil, "30.00"},
		{"200", "", "7.5", nil, "15.00"},
		{"19.99", "", "7", nil, "1.40"},
		{"19.99", "", "7", []Rounding{Floor}, "1.39"},
		{"1000", "JPY", "8.25", nil, "83"},
		{"10", "KWD", "0.125", nil, "0.013"},
	}
	for _, tt := range tests {
		t.Run(tt.amount+" "+tt.currency, func(t *testing.T) {
			assert.Equal(t, tt.want, must(t)(of(t, tt.amount, tt.currency).Percent(tt.percent, tt.rounding...)).Decimal())
		})
	}
}

func TestAddAndSubtractPercent(t *testing.T) {
	assert.Equal(t, "107.00", must(t)(of(t, "100", "").AddPercent(7)).Decimal())
	assert.Equal(t, "21.76", must(t)(of(t, "19.99", "").AddPercent("8.875")).Decimal())
	assert.Equal(t, "85.00", must(t)(of(t, "100", "").SubtractPercent(15)).Decimal())
	assert.Equal(t, "6.66", must(t)(of(t, "9.99", "").SubtractPercent("33.333", Ceiling)).Decimal())

	_, err := of(t, "1", "").AddPercent("x")
	assert.ErrorIs(t, err, ErrInvalidDecimal)
	_, err = of(t, "1", "").SubtractPercent(1.5)
	assert.ErrorIs(t, err, ErrInvalidOperand)
}

func TestPercentageOf(t *testing.T) {
	tests := []struct {
		amount, total, currency string
		scale                   int
		rounding                []Rounding
		want                    string
	}{
		{"25", "200", "", 2, nil, "12.50"},
		{"1", "3", "", 2, nil, "33.33"},
		{"2", "3", "", 2, nil, "66.67"},
		{"2", "3", "", 4, nil, "66.6667"},
		{"2", "3", "", 0, []Rounding{Floor}, "66"},
		{"300", "200", "", 2, nil, "150.00"},
		{"-50", "200", "JPY", 2, nil, "-25.00"},
	}
	for _, tt := range tests {
		t.Run(tt.amount+"/"+tt.total, func(t *testing.T) {
			got, err := of(t, tt.amount, tt.currency).PercentageOf(of(t, tt.total, tt.currency), tt.scale, tt.rounding...)
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}

	_, err := of(t, "1", "").PercentageOf(of(t, "0", ""), 2)
	assert.ErrorIs(t, err, ErrDivisionByZero)
	_, err = of(t, "1", "").PercentageOf(of(t, "1", "EUR"), 2)
	assert.ErrorIs(t, err, ErrCurrencyMismatch)
	_, err = of(t, "1", "").PercentageOf(of(t, "1", ""), -1)
	assert.ErrorIs(t, err, ErrInvalidOperand)
}

func TestRatioOf(t *testing.T) {
	tests := []struct {
		amount, other string
		scale         int
		rounding      []Rounding
		want          string
	}{
		{"50", "200", 4, nil, "0.2500"},
		{"1", "3", 6, nil, "0.333333"},
		{"2", "3", 2, []Rounding{Floor}, "0.66"},
	}
	for _, tt := range tests {
		t.Run(tt.amount+"/"+tt.other, func(t *testing.T) {
			got, err := of(t, tt.amount, "").RatioOf(of(t, tt.other, ""), tt.scale, tt.rounding...)
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}

	_, err := of(t, "1", "").RatioOf(of(t, "0", ""), 4)
	assert.ErrorIs(t, err, ErrDivisionByZero)
	_, err = of(t, "1", "").RatioOf(of(t, "1", "EUR"), 4)
	assert.ErrorIs(t, err, ErrCurrencyMismatch)
}

func TestSplit(t *testing.T) {
	parts, err := of(t, "100.00", "").Split(3)
	require.NoError(t, err)
	assert.Equal(t, []string{"33.34", "33.33", "33.33"}, decimals(parts))
	assert.Equal(t, "100.00", must(t)(Sum(parts)).Decimal())

	tests := []struct {
		amount, currency string
		parts            int
		want             []string
	}{
		{"1000", "JPY", 3, []string{"334", "333", "333"}},
		{"1", "KWD", 6, []string{"0.167", "0.167", "0.167", "0.167", "0.166", "0.166"}},
		{"-100", "", 3, []string{"-33.34", "-33.33", "-33.33"}},
		{"5", "", 1, []string{"5.00"}},
	}
	for _, tt := range tests {
		t.Run(tt.amount+" "+tt.currency, func(t *testing.T) {
			parts, err := of(t, tt.amount, tt.currency).Split(tt.parts)
			require.NoError(t, err)
			assert.Equal(t, tt.want, decimals(parts))
		})
	}
}

func TestAllocate(t *testing.T) {
	shares, err := AllocateMap(of(t, "100.00", ""), map[string]int{"owner": 70, "agent": 20, "platform": 10})
	require.NoError(t, err)
	assert.Equal(t, "70.00", shares["owner"].Decimal())
	assert.Equal(t, "20.00", shares["agent"].Decimal())
	assert.Equal(t, "10.00", shares["platform"].Decimal())
	assert.Len(t, shares, 3)
}

func TestAllocateGivesLeftoverUnitsToTheLargestRemainders(t *testing.T) {
	tests := []struct {
		name   string
		minor  string
		ratios []any
		want   []string
	}{
		{"remainders", "5", []any{3, 7}, []string{"2", "3"}},
		{"decimal ratios", "10", []any{"0.3", "0.3", "0.4"}, []string{"3", "3", "4"}},
		{"ties go to earlier parts", "2", []any{1, 1, 1}, []string{"1", "1", "0"}},
		{"zero ratio", "100", []any{0, 1}, []string{"0", "100"}},
		{"moneyphp USD(1001) [3,5,11,7]", "1001", []any{3, 5, 11, 7}, []string{"116", "193", "423", "269"}},
		{"mixed scales", "1000", []any{"0.25", 1, "1.125"}, []string{"105", "421", "474"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			shares, err := minor(t, tt.minor, "").Allocate(tt.ratios...)
			require.NoError(t, err)
			assert.Equal(t, tt.want, amounts(shares))
		})
	}
}

func TestAllocateMapIsDeterministic(t *testing.T) {
	type key string
	for range 20 {
		shares, err := AllocateMap(minor(t, "2", ""), map[key]string{"c": "1", "a": "1", "b": "1"})
		require.NoError(t, err)
		assert.Equal(t, "1", shares["a"].Amount(), "ties go to keys in ascending order")
		assert.Equal(t, "1", shares["b"].Amount())
		assert.Equal(t, "0", shares["c"].Amount())
	}

	byID, err := AllocateMap(minor(t, "-5", ""), map[int]uint{2: 7, 1: 3})
	require.NoError(t, err)
	assert.Equal(t, "-2", byID[1].Amount())
	assert.Equal(t, "-3", byID[2].Amount())

	_, err = AllocateMap(minor(t, "5", ""), map[string]int{})
	assert.ErrorIs(t, err, ErrInvalidAllocation)
}

func TestAllocateRejectsInvalidSplitsAndRatios(t *testing.T) {
	one := of(t, "1", "")
	tests := []struct {
		name    string
		call    func() error
		message string
	}{
		{"split zero", func() error { _, err := one.Split(0); return err }, "money: money can only be split into one or more parts, 0 given"},
		{"no ratios", func() error { _, err := one.Allocate(); return err }, "money: cannot allocate money: at least one ratio is required"},
		{"negative ratio", func() error { _, err := one.Allocate(1, -1); return err }, "money: cannot allocate money: ratios must be zero or positive"},
		{"zero sum", func() error { _, err := one.Allocate(0, "0.0"); return err }, "money: cannot allocate money: the sum of the ratios must be greater than zero"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.call()
			require.ErrorIs(t, err, ErrInvalidAllocation)
			assert.EqualError(t, err, tt.message)
		})
	}

	_, err := one.Allocate("abc")
	assert.ErrorIs(t, err, ErrInvalidDecimal)
}

func TestRoundTo(t *testing.T) {
	tests := []struct {
		amount, currency string
		decimals         int
		rounding         []Rounding
		want             string
	}{
		{"12.34", "", 0, nil, "12.00"},
		{"12.50", "", 0, nil, "13.00"},
		{"12.50", "", 0, []Rounding{HalfEven}, "12.00"},
		{"12.34", "", 1, []Rounding{Ceiling}, "12.40"},
		{"15", "JPY", -1, nil, "20"},
		{"12.345", "KWD", 2, nil, "12.350"},
		{"12.34", "", 2, nil, "12.34"},
		{"12.34", "", 5, nil, "12.34"},
	}
	for _, tt := range tests {
		t.Run(tt.amount, func(t *testing.T) {
			assert.Equal(t, tt.want, of(t, tt.amount, tt.currency).RoundTo(tt.decimals, tt.rounding...).Decimal())
		})
	}
}

func TestEveryRoundingModeOnTiesAndNonTies(t *testing.T) {
	inputs := []string{"2.5", "3.5", "-2.5", "-3.5", "2.4", "-2.6", "2.1", "-2.1"}
	tests := []struct {
		rounding Rounding
		want     []string
	}{
		{HalfUp, []string{"3", "4", "-3", "-4", "2", "-3", "2", "-2"}},
		{HalfDown, []string{"2", "3", "-2", "-3", "2", "-3", "2", "-2"}},
		{HalfEven, []string{"2", "4", "-2", "-4", "2", "-3", "2", "-2"}},
		{HalfOdd, []string{"3", "3", "-3", "-3", "2", "-3", "2", "-2"}},
		{HalfPositiveInfinity, []string{"3", "4", "-2", "-3", "2", "-3", "2", "-2"}},
		{HalfNegativeInfinity, []string{"2", "3", "-3", "-4", "2", "-3", "2", "-2"}},
		{Ceiling, []string{"3", "4", "-2", "-3", "3", "-2", "3", "-2"}},
		{Floor, []string{"2", "3", "-3", "-4", "2", "-3", "2", "-3"}},
	}
	for _, tt := range tests {
		t.Run(tt.rounding.String(), func(t *testing.T) {
			got := make([]string, len(inputs))
			for i, input := range inputs {
				got[i] = of(t, input, "JPY", tt.rounding).Amount()
			}
			assert.Equal(t, tt.want, got)
		})
	}
}

// TestRoundingMatchesMoneyPHP uses moneyphp's results for
// JPY(10)->multiply($value)->divide('10'), which laravel-money's suite
// compares against.
func TestRoundingMatchesMoneyPHP(t *testing.T) {
	inputs := []string{"2.5", "3.5", "-2.5", "-3.5", "2.4", "-2.6", "0.5", "-0.5"}
	want := map[Rounding][]string{
		HalfUp:               {"3", "4", "-3", "-4", "2", "-3", "1", "-1"},
		HalfDown:             {"2", "3", "-2", "-3", "2", "-3", "0", "0"},
		HalfEven:             {"2", "4", "-2", "-4", "2", "-3", "0", "0"},
		HalfOdd:              {"3", "3", "-3", "-3", "2", "-3", "1", "-1"},
		HalfPositiveInfinity: {"3", "4", "-2", "-3", "2", "-3", "1", "0"},
		HalfNegativeInfinity: {"2", "3", "-3", "-4", "2", "-3", "0", "-1"},
		Ceiling:              {"3", "4", "-2", "-3", "3", "-2", "1", "0"},
		Floor:                {"2", "3", "-3", "-4", "2", "-3", "0", "-1"},
	}
	for _, rounding := range Roundings() {
		t.Run(rounding.String(), func(t *testing.T) {
			got := make([]string, len(inputs))
			for i, input := range inputs {
				got[i] = must(t)(minor(t, "10", "JPY").Times(input, rounding)).Amount()
				got[i] = must(t)(minor(t, got[i], "JPY").DividedBy("10", rounding)).Amount()
			}
			assert.Equal(t, want[rounding], got)
		})
	}
}

func TestTheConfiguredDefaultRounding(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Rounding = Floor
	useConfig(t, cfg)

	assert.Equal(t, "3.33", must(t)(of(t, "10", "").DividedBy(3)).Decimal())
	assert.Equal(t, "6.66", must(t)(of(t, "20", "").DividedBy(3)).Decimal())
	assert.Equal(t, "12.00", of(t, "12.99", "").RoundTo(0).Decimal())

	cfg.Rounding = Ceiling
	useConfig(t, cfg)
	assert.Equal(t, "3.34", must(t)(of(t, "10", "").DividedBy(3)).Decimal())

	_, err := ParseRounding("sideways")
	require.ErrorIs(t, err, ErrInvalidConfig)
	assert.ErrorContains(t, err, `money: the config value money.rounding must be one of "half_up", "half_down"`)
}
