package money

import (
	"strconv"
	"strings"
)

// Rounding says how an amount that falls between two minor units is rounded.
//
// The "half" modes only differ when the amount is exactly halfway; every other
// amount rounds to the nearest minor unit. The zero value is HalfUp, the
// default of the money.rounding config value.
type Rounding int

const (
	// HalfUp rounds halfway values away from zero: 2.5 -> 3, -2.5 -> -3. The default.
	HalfUp Rounding = iota
	// HalfDown rounds halfway values toward zero: 2.5 -> 2, -2.5 -> -2.
	HalfDown
	// HalfEven rounds halfway values to the even neighbor (banker's rounding): 2.5 -> 2, 3.5 -> 4.
	HalfEven
	// HalfOdd rounds halfway values to the odd neighbor: 2.5 -> 3, 3.5 -> 3.
	HalfOdd
	// HalfPositiveInfinity rounds halfway values toward positive infinity: 2.5 -> 3, -2.5 -> -2.
	HalfPositiveInfinity
	// HalfNegativeInfinity rounds halfway values toward negative infinity: 2.5 -> 2, -2.5 -> -3.
	HalfNegativeInfinity
	// Ceiling always rounds toward positive infinity: 2.1 -> 3, -2.9 -> -2.
	Ceiling
	// Floor always rounds toward negative infinity: 2.9 -> 2, -2.1 -> -3.
	Floor
)

var roundingNames = [...]string{
	HalfUp:               "half_up",
	HalfDown:             "half_down",
	HalfEven:             "half_even",
	HalfOdd:              "half_odd",
	HalfPositiveInfinity: "half_positive_infinity",
	HalfNegativeInfinity: "half_negative_infinity",
	Ceiling:              "ceiling",
	Floor:                "floor",
}

// Roundings returns every rounding mode.
func Roundings() []Rounding {
	return []Rounding{HalfUp, HalfDown, HalfEven, HalfOdd, HalfPositiveInfinity, HalfNegativeInfinity, Ceiling, Floor}
}

// ParseRounding returns the mode for a config value such as "half_even".
func ParseRounding(name string) (Rounding, error) {
	normalized := strings.ToLower(strings.TrimSpace(name))
	for mode, n := range roundingNames {
		if n == normalized {
			return Rounding(mode), nil
		}
	}

	return HalfUp, invalidConfig("rounding", `one of "`+strings.Join(roundingNames[:], `", "`)+`"`)
}

// String returns the config value of the mode, for example "half_up".
func (r Rounding) String() string {
	if r < 0 || int(r) >= len(roundingNames) {
		return "rounding(" + strconv.Itoa(int(r)) + ")"
	}

	return roundingNames[r]
}

// MarshalText encodes the mode as its config value.
func (r Rounding) MarshalText() ([]byte, error) {
	if r < 0 || int(r) >= len(roundingNames) {
		return nil, invalidConfig("rounding", "a valid rounding mode")
	}

	return []byte(r.String()), nil
}

// UnmarshalText decodes a config value such as "floor".
func (r *Rounding) UnmarshalText(text []byte) error {
	mode, err := ParseRounding(string(text))
	if err != nil {
		return err
	}
	*r = mode

	return nil
}

// validate returns ErrInvalidOperand for a value that is not one of the
// eight modes, such as Rounding(42), instead of silently rounding half up.
func (r Rounding) validate() error {
	if r < HalfUp || r > Floor {
		return newError(ErrInvalidOperand, "%s is not a rounding mode; use one of money.Roundings()", r)
	}

	return nil
}

// pick returns the first given rounding, or the configured default. A given
// rounding that is not a valid mode returns ErrInvalidOperand.
func pick(rounding []Rounding) (Rounding, error) {
	if len(rounding) > 0 {
		return rounding[0], rounding[0].validate()
	}
	manager, err := current()
	if err != nil {
		return HalfUp, err
	}

	return manager.Rounding(), nil
}
