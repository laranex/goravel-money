package money

import (
	"math/big"
	"reflect"
	"strconv"
	"strings"
	"unicode/utf8"
)

// Exact decimal arithmetic on integers (math/big). Every value is an integer
// or a fraction of two integers, so nothing ever passes through a float. This
// is the Go port of laravel-money's Support\Decimal.

var (
	bigOne     = big.NewInt(1)
	bigTwo     = big.NewInt(2)
	bigHundred = big.NewInt(100)
)

// parsedDecimal is a decimal amount split into sign, integer and fraction digits.
type parsedDecimal struct {
	negative bool
	integer  string // digits without leading zeros, "0" for zero
	fraction string // fraction digits as given, possibly empty
}

const groupSeparators = ", \u00A0\u202F"

// parseDecimal parses a decimal string such as "1234.50", "-0.5", "+5" or
// "1,234.50". The integer part may be grouped with one separator (comma,
// space, no-break space or narrow no-break space) used consistently: Western
// groups of three (1,234,567) or Indian grouping (12,34,567). The decimal
// separator is always a dot.
func parseDecimal(input string) (parsedDecimal, error) {
	body := strings.Trim(input, " \t\n\r\x00\x0B")
	negative := false
	if body != "" && (body[0] == '-' || body[0] == '+') {
		negative = body[0] == '-'
		body = body[1:]
	}

	parts := strings.Split(body, ".")
	if len(parts) > 2 {
		return parsedDecimal{}, invalidDecimal(input)
	}
	integer := parts[0]
	fraction := ""
	if len(parts) == 2 {
		fraction = parts[1]
	}

	if strings.ContainsAny(integer, groupSeparators) {
		if !validGrouping(integer) {
			return parsedDecimal{}, invalidDecimal(input)
		}
		integer = strings.NewReplacer(",", "", " ", "", "\u00A0", "", "\u202F", "").Replace(integer)
	}
	if !isDigits(integer) || (len(parts) == 2 && !isDigits(fraction)) {
		return parsedDecimal{}, invalidDecimal(input)
	}

	integer = trimZeros(integer)

	return parsedDecimal{
		negative: negative && (integer != "0" || strings.Trim(fraction, "0") != ""),
		integer:  integer,
		fraction: fraction,
	}, nil
}

// validGrouping reports whether integer is digits grouped by a single
// separator used throughout: Western groups of three (1,234,567) or Indian
// grouping (12,34,567: the last group has three digits, earlier groups two).
// It accepts exactly what laravel-money accepts.
func validGrouping(integer string) bool {
	sep := strings.IndexAny(integer, groupSeparators)
	if sep < 0 {
		return false
	}
	separator, _ := utf8.DecodeRuneInString(integer[sep:])
	groups := strings.Split(integer, string(separator))
	if len(groups) < 2 {
		return false
	}
	for _, group := range groups {
		if !isDigits(group) {
			return false // empty, another separator or a non-digit
		}
	}

	first, last, middle := len(groups[0]), len(groups[len(groups)-1]), groups[1:len(groups)-1]
	if last != 3 || first > 3 {
		return false
	}
	size := 3 // Western; Indian when the second group has two digits
	if len(middle) > 0 && len(middle[0]) == 2 {
		size = 2
		if first > 2 {
			return false
		}
	}
	for _, group := range middle {
		if len(group) != size {
			return false
		}
	}

	return true
}

func isDigits(s string) bool {
	if s == "" {
		return false
	}
	for i := 0; i < len(s); i++ {
		if s[i] < '0' || s[i] > '9' {
			return false
		}
	}

	return true
}

func trimZeros(digits string) string {
	digits = strings.TrimLeft(digits, "0")
	if digits == "" {
		return "0"
	}

	return digits
}

func invalidDecimal(input string) error {
	return &ParseError{Input: input, Err: ErrInvalidDecimal}
}

// toMinor turns a decimal amount into minor units with the given precision.
// Extra fraction digits that are all zeros are dropped; any other extra
// digits return a *ParseError unless a rounding mode is given.
func toMinor(input string, precision int, rounding *Rounding, currency string) (*big.Int, error) {
	parsed, err := parseDecimal(input)
	if err != nil {
		if pe, ok := err.(*ParseError); ok {
			pe.Currency, pe.Precision = currency, precision
		}

		return nil, err
	}
	sign := ""
	if parsed.negative {
		sign = "-"
	}
	fraction := parsed.fraction

	if len(fraction) <= precision {
		return mustBig(sign + parsed.integer + fraction + strings.Repeat("0", precision-len(fraction))), nil
	}
	extra := fraction[precision:]
	if strings.Trim(extra, "0") == "" {
		return mustBig(sign + parsed.integer + fraction[:precision]), nil
	}
	if rounding == nil {
		return nil, &ParseError{Input: strings.Trim(input, " \t\n\r\x00\x0B"), Currency: currency, Precision: precision, Decimals: len(fraction), Err: ErrTooManyDecimals}
	}

	return divide(mustBig(sign+parsed.integer+fraction), pow10(len(extra)), *rounding)
}

// fromMinor formats minor units as a decimal string with exactly precision decimals.
func fromMinor(minor *big.Int, precision int) string {
	digits := new(big.Int).Abs(minor).String()
	if precision > 0 {
		if len(digits) < precision+1 {
			digits = strings.Repeat("0", precision+1-len(digits)) + digits
		}
		digits = digits[:len(digits)-precision] + "." + digits[len(digits)-precision:]
	}
	if minor.Sign() < 0 {
		return "-" + digits
	}

	return digits
}

// fraction returns a decimal number as an exact fraction: numerator and a
// power-of-ten denominator.
func fraction(input string) (*big.Int, *big.Int, error) {
	parsed, err := parseDecimal(input)
	if err != nil {
		return nil, nil, err
	}
	sign := ""
	if parsed.negative {
		sign = "-"
	}

	return mustBig(sign + parsed.integer + parsed.fraction), pow10(len(parsed.fraction)), nil
}

// divide divides two integers and rounds the quotient to an integer, exactly.
func divide(numerator, denominator *big.Int, rounding Rounding) (*big.Int, error) {
	if denominator.Sign() == 0 {
		return nil, divisionByZero()
	}
	num, den := new(big.Int).Set(numerator), new(big.Int).Set(denominator)
	if den.Sign() < 0 {
		num.Neg(num)
		den.Neg(den)
	}

	quotient, remainder := new(big.Int).QuoRem(num, den, new(big.Int))
	if remainder.Sign() == 0 {
		return quotient, nil
	}

	positive := num.Sign() > 0
	half := new(big.Int).Mul(new(big.Int).Abs(remainder), bigTwo).Cmp(den)
	odd := quotient.Bit(0) == 1

	var away bool
	switch rounding {
	case Ceiling:
		away = positive
	case Floor:
		away = !positive
	default:
		away = half > 0
		if half == 0 {
			switch rounding {
			case HalfDown:
				away = false
			case HalfEven:
				away = odd
			case HalfOdd:
				away = !odd
			case HalfPositiveInfinity:
				away = positive
			case HalfNegativeInfinity:
				away = !positive
			default: // HalfUp
				away = true
			}
		}
	}
	if !away {
		return quotient, nil
	}
	if positive {
		return quotient.Add(quotient, bigOne), nil
	}

	return quotient.Sub(quotient, bigOne), nil
}

// quotient divides two integers and returns a decimal string with scale decimals.
func quotient(numerator, denominator *big.Int, scale int, rounding Rounding) (string, error) {
	q, err := divide(new(big.Int).Mul(numerator, pow10(scale)), denominator, rounding)
	if err != nil {
		return "", err
	}

	return fromMinor(q, scale), nil
}

func pow10(exponent int) *big.Int {
	if exponent <= 0 {
		return big.NewInt(1)
	}

	return new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(exponent)), nil)
}

func mustBig(digits string) *big.Int {
	n, ok := new(big.Int).SetString(digits, 10)
	if !ok {
		panic("money: internal error: " + strconv.Quote(digits) + " is not an integer")
	}

	return n
}

// numberString turns a multiplier, ratio or operand into a decimal string.
// Strings (including named string types such as json.Number), integers and
// *big.Int are accepted; floats are rejected because they are not exact.
func numberString(value any) (string, error) {
	switch v := value.(type) {
	case string:
		return v, nil
	case *big.Int:
		if v == nil {
			return "", unsupportedOperand(value)
		}

		return v.String(), nil
	case nil:
		return "", unsupportedOperand(value)
	}

	rv := reflect.ValueOf(value)
	switch rv.Kind() {
	case reflect.String:
		return rv.String(), nil
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return strconv.FormatInt(rv.Int(), 10), nil
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return strconv.FormatUint(rv.Uint(), 10), nil
	case reflect.Float32, reflect.Float64:
		return "", floatOperand(value)
	default:
		return "", unsupportedOperand(value)
	}
}

// numberFraction parses a multiplier or ratio into an exact fraction.
func numberFraction(value any) (*big.Int, *big.Int, error) {
	text, err := numberString(value)
	if err != nil {
		return nil, nil, err
	}

	return fraction(text)
}
