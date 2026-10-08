package money

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRoundingNames(t *testing.T) {
	names := map[Rounding]string{
		HalfUp: "half_up", HalfDown: "half_down", HalfEven: "half_even", HalfOdd: "half_odd",
		HalfPositiveInfinity: "half_positive_infinity", HalfNegativeInfinity: "half_negative_infinity",
		Ceiling: "ceiling", Floor: "floor",
	}
	require.Len(t, Roundings(), 8)
	for _, mode := range Roundings() {
		t.Run(names[mode], func(t *testing.T) {
			assert.Equal(t, names[mode], mode.String())

			parsed, err := ParseRounding(" " + names[mode] + " ")
			require.NoError(t, err)
			assert.Equal(t, mode, parsed)

			text, err := mode.MarshalText()
			require.NoError(t, err)
			var decoded Rounding
			require.NoError(t, decoded.UnmarshalText(text))
			assert.Equal(t, mode, decoded)
		})
	}
	assert.Equal(t, HalfUp, Rounding(0), "the zero value is the default")
}

func TestRoundingRejectsUnknownModes(t *testing.T) {
	assert.Equal(t, "rounding(42)", Rounding(42).String())
	_, err := Rounding(-1).MarshalText()
	assert.ErrorIs(t, err, ErrInvalidConfig)

	var r Rounding
	assert.ErrorIs(t, r.UnmarshalText([]byte("up")), ErrInvalidConfig)

	cfg := DefaultConfig()
	cfg.Rounding = Rounding(99)
	_, err = NewManager(cfg)
	assert.ErrorIs(t, err, ErrInvalidConfig)
}
