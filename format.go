package money

import (
	"sort"
	"strings"
	"unicode"
	"unicode/utf8"
)

// localeFormat is the currency format of one locale, generated from ICU into
// formats_cldr.go.
type localeFormat struct {
	decimal, group              string
	primary, secondary, minimum int
	zero                        rune
	// affixes are the positive prefix, positive suffix, negative prefix and
	// negative suffix, with "¤" standing for the currency symbol.
	affixes   [4]string
	symbols   map[string]string    // currency code => symbol, where it differs from the code
	overrides map[string][4]string // exact affixes where the template does not apply
}

// Locales returns the locales Format supports, sorted. Other locales fall back
// to their language ("de_LU" uses "de"), then to the plain "USD 1234.50".
func Locales() []string {
	list := make([]string, 0, len(cldrLocales))
	for locale := range cldrLocales {
		list = append(list, locale)
	}
	sort.Strings(list)

	return list
}

// FormatDecimal formats a decimal string such as "-1234.50" (as returned by
// Money.Decimal) in a currency and locale. It is exact for amounts of any size:
// the locale's symbol, separators, grouping (including Indian lakh grouping)
// and digits come from CLDR data generated from ICU, and the number is built
// from the decimal string, never from a float. An empty or unsupported locale
// returns "USD 1234.50".
func FormatDecimal(decimal string, currency string, locale string) string {
	format := lookupLocale(locale)
	if format == nil {
		return currency + " " + decimal
	}

	negative := strings.HasPrefix(decimal, "-")
	integer, fraction, hasFraction := strings.Cut(strings.TrimPrefix(decimal, "-"), ".")

	var b strings.Builder
	affixes := format.affixesFor(currency)
	if negative {
		b.WriteString(affixes[2])
	} else {
		b.WriteString(affixes[0])
	}
	for i, group := range format.groups(integer) {
		if i > 0 {
			b.WriteString(format.group)
		}
		format.writeDigits(&b, group)
	}
	if hasFraction {
		b.WriteString(format.decimal)
		format.writeDigits(&b, fraction)
	}
	if negative {
		b.WriteString(affixes[3])
	} else {
		b.WriteString(affixes[1])
	}

	return b.String()
}

func lookupLocale(locale string) *localeFormat {
	if i := strings.IndexAny(locale, ".@"); i >= 0 {
		locale = locale[:i]
	}
	language, region, _ := strings.Cut(strings.ReplaceAll(strings.TrimSpace(locale), "-", "_"), "_")
	language = strings.ToLower(language)
	if language == "" {
		return nil
	}
	if region != "" {
		if f, ok := cldrLocales[language+"_"+strings.ToUpper(region)]; ok {
			return f
		}
	}

	return cldrLocales[language]
}

func (f *localeFormat) affixesFor(currency string) [4]string {
	if affixes, ok := f.overrides[currency]; ok {
		return affixes
	}
	symbol, ok := f.symbols[currency]
	if !ok {
		symbol = currency
	}

	return [4]string{
		renderAffix(f.affixes[0], symbol, true),
		renderAffix(f.affixes[1], symbol, false),
		renderAffix(f.affixes[2], symbol, true),
		renderAffix(f.affixes[3], symbol, false),
	}
}

// renderAffix replaces "¤" with the symbol and applies CLDR currency spacing:
// a no-break space separates a symbol that touches the number when the
// symbol's character next to the number is not a symbol or space ("KWD 1.000"
// but "$1.00").
func renderAffix(template, symbol string, prefix bool) string {
	if !strings.Contains(template, "¤") {
		return template
	}
	out := strings.Replace(template, "¤", symbol, 1)
	if prefix && strings.HasSuffix(template, "¤") {
		if r, _ := utf8.DecodeLastRuneInString(symbol); !isSymbolOrSpace(r) {
			out += "\u00A0"
		}
	}
	if !prefix && strings.HasPrefix(template, "¤") {
		if r, _ := utf8.DecodeRuneInString(symbol); !isSymbolOrSpace(r) {
			out = "\u00A0" + out
		}
	}

	return out
}

func isSymbolOrSpace(r rune) bool {
	return unicode.In(r, unicode.S, unicode.Z)
}

// groups splits integer digits by the locale's grouping sizes.
func (f *localeFormat) groups(integer string) []string {
	if f.primary <= 0 || len(integer) < f.primary+f.minimum {
		return []string{integer}
	}
	groups := []string{integer[len(integer)-f.primary:]}
	rest := integer[:len(integer)-f.primary]
	for len(rest) > f.secondary {
		groups = append(groups, rest[len(rest)-f.secondary:])
		rest = rest[:len(rest)-f.secondary]
	}
	if rest != "" {
		groups = append(groups, rest)
	}
	for i, j := 0, len(groups)-1; i < j; i, j = i+1, j-1 {
		groups[i], groups[j] = groups[j], groups[i]
	}

	return groups
}

func (f *localeFormat) writeDigits(b *strings.Builder, digits string) {
	if f.zero == '0' {
		b.WriteString(digits)

		return
	}
	for i := 0; i < len(digits); i++ {
		b.WriteRune(f.zero + rune(digits[i]-'0'))
	}
}
