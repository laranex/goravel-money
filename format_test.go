package money

import (
	"bufio"
	"os"
	"sort"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestFormatMatchesICU compares every vector in testdata/formats.tsv, which
// scripts/generate-formats.php writes from ICU's own output for every
// supported locale and a spread of currencies (symbols, codes, 0 to 4
// decimals, a custom code) and amounts.
func TestFormatMatchesICU(t *testing.T) {
	file, err := os.Open("testdata/formats.tsv")
	require.NoError(t, err)
	defer file.Close()

	count := 0
	locales := map[string]bool{}
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "#") || line == "" {
			continue
		}
		fields := strings.Split(line, "\t")
		require.Len(t, fields, 5, line)
		locale, code, decimal, want := fields[0], fields[1], fields[3], fields[4]
		precision, err := strconv.Atoi(fields[2])
		require.NoError(t, err)

		currency, err := NewCurrency(code, precision)
		require.NoError(t, err)
		m := MustParse(decimal, currency)
		require.Equal(t, decimal, m.Decimal())

		assert.Equal(t, want, m.Format(locale), "%s %s %s", locale, code, decimal)
		locales[locale] = true
		count++
	}
	require.NoError(t, scanner.Err())
	assert.Greater(t, count, 8000)
	assert.Len(t, locales, len(cldrLocales), "every supported locale has vectors")
}

func TestFormatWithoutALocaleIsPlain(t *testing.T) {
	assert.Equal(t, "USD 1234.50", of(t, "1234.5", "USD").Format())
	assert.Equal(t, "KWD -3.000", of(t, "-3", "KWD").Format(""))
	assert.Equal(t, "JPY 15", of(t, "15", "JPY").Format())
	assert.Equal(t, "USD 1.00", of(t, "1", "USD").Format("xx_YY"), "unsupported locales are plain")
}

// TestFormatLaravelMoneyExamples are laravel-money's FormattingTest vectors.
func TestFormatLaravelMoneyExamples(t *testing.T) {
	useConfig(t, laravelConfig())

	tests := []struct {
		amount, currency, locale, want string
	}{
		{"1234.5", "USD", "en", "$1,234.50"},
		{"-1234.5", "USD", "en", "-$1,234.50"},
		{"1500", "JPY", "en", "¥1,500"},
		{"1.5", "KWD", "en", "KWD\u00A01.500"},
		{"300", "PTS", "en", "PTS\u00A0300"},
		{"1234.5", "EUR", "de_DE", "1.234,50\u00A0€"},
		{"1234567.89", "INR", "en_IN", "₹12,34,567.89"},
		{"1234.5", "EUR", "fr_FR", "1\u202F234,50\u00A0€"},
		{"12345.5", "EUR", "es_ES", "12.345,50\u00A0€"},
		{"1234.5", "EUR", "es_ES", "1.234,50\u00A0€"},
		{"123456789012345678901.23", "USD", "en", "$123,456,789,012,345,678,901.23"},
		{"90071992547409.93", "USD", "en", "$90,071,992,547,409.93"},
		{"1500", "MMK", "my_MM", "၁,၅၀၀.၀၀\u00A0K"},
	}
	for _, tt := range tests {
		t.Run(tt.locale+" "+tt.amount, func(t *testing.T) {
			assert.Equal(t, tt.want, of(t, tt.amount, tt.currency).Format(tt.locale))
		})
	}
}

func TestFormatFollowsTheConfiguredLocale(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Locale = "de_DE"
	manager := useConfig(t, cfg)

	m := of(t, "1234.5", "EUR")
	assert.Equal(t, "1.234,50\u00A0€", m.Format())
	assert.Equal(t, "1.234,50\u00A0€", manager.Format(m))
	assert.Equal(t, "€1,234.50", manager.Format(m, "en_US"), "an explicit locale wins")
	assert.Equal(t, "EUR 1234.50", manager.Format(m, ""), "an explicit empty locale is plain")
	assert.Equal(t, "EUR 1234.50", m.String(), "String never depends on config")
}

func TestFormatNormalizesLocales(t *testing.T) {
	m := of(t, "1234.5", "EUR")
	want := "1.234,50\u00A0€"
	for _, locale := range []string{"de_DE", "de-DE", "DE_de", "de_DE.UTF-8", "de_DE@euro", " de_DE ", "de", "de_LU"} {
		t.Run(locale, func(t *testing.T) {
			assert.Equal(t, want, m.Format(locale))
		})
	}
	assert.Equal(t, "€\u00A01.234,50", m.Format("de_AT"), "a supported region wins over the language")
}

func TestFormatDecimal(t *testing.T) {
	assert.Equal(t, "$0.05", FormatDecimal("0.05", "USD", "en"))
	assert.Equal(t, "-¥1", FormatDecimal("-1", "JPY", "en"))
	assert.Equal(t, "GBP 1.00", FormatDecimal("1.00", "GBP", ""))
}

func TestLocales(t *testing.T) {
	locales := Locales()
	assert.Len(t, locales, len(cldrLocales))
	assert.True(t, sort.StringsAreSorted(locales))
	assert.Contains(t, locales, "my_MM")
	assert.Equal(t, "77.1", icuVersion)
}

func TestRenderAffixCurrencySpacing(t *testing.T) {
	tests := []struct {
		template, symbol string
		prefix           bool
		want             string
	}{
		{"¤", "$", true, "$"},
		{"¤", "KWD", true, "KWD\u00A0"},
		{"-¤", "KWD", true, "-KWD\u00A0"},
		{"¤-", "KWD", true, "KWD-"},
		{"¤", "€", false, "€"},
		{"¤", "KWD", false, "\u00A0KWD"},
		{"\u00A0¤", "KWD", false, "\u00A0KWD"},
		{"-", "KWD", true, "-"},
	}
	for _, tt := range tests {
		t.Run(tt.template+tt.symbol, func(t *testing.T) {
			assert.Equal(t, tt.want, renderAffix(tt.template, tt.symbol, tt.prefix))
		})
	}
}

func TestOverridesAreUsedVerbatim(t *testing.T) {
	format := &localeFormat{decimal: ".", group: ",", primary: 3, secondary: 3, minimum: 1, zero: '0',
		affixes:   [4]string{"¤", "", "-¤", ""},
		overrides: map[string][4]string{"XTS": {"<", ">", "(", ")"}},
	}
	assert.Equal(t, [4]string{"<", ">", "(", ")"}, format.affixesFor("XTS"))
	assert.Equal(t, [4]string{"ABC\u00A0", "", "-ABC\u00A0", ""}, format.affixesFor("ABC"))
}
