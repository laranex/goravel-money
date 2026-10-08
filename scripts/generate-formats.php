<?php

/*
 * Generates formats_cldr.go and testdata/formats.tsv from ICU (PHP's intl
 * extension), so the Go formatter can reproduce ICU's currency formatting
 * exactly without ICU and without floats.
 *
 *     php scripts/generate-formats.php
 *
 * For every supported locale it records the monetary decimal and grouping
 * separators, the grouping sizes, the digits, the positive and negative
 * affix templates ("¤" is the currency symbol) and the currency symbols that
 * differ from the ISO code. It then rebuilds the affixes of every ISO 4217
 * currency from that model with the same rules as format.go and stores an
 * explicit override wherever the model does not match ICU, so the generated
 * data is exact by construction.
 */

declare(strict_types=1);

const LOCALES = [
    'en', 'en_US', 'en_GB', 'en_AU', 'en_CA', 'en_IN', 'en_NZ', 'en_SG',
    'de', 'de_DE', 'de_AT', 'de_CH',
    'fr', 'fr_FR', 'fr_CA', 'fr_CH',
    'es', 'es_ES', 'es_MX',
    'it', 'it_IT', 'nl', 'nl_NL', 'pt', 'pt_BR', 'pt_PT',
    'pl', 'pl_PL', 'ru', 'ru_RU', 'tr', 'tr_TR', 'sv', 'sv_SE',
    'ja', 'ja_JP', 'ko', 'ko_KR', 'zh', 'zh_CN', 'zh_TW', 'zh_HK',
    'th', 'th_TH', 'vi', 'vi_VN', 'id', 'id_ID', 'ms', 'ms_MY',
    'my', 'my_MM', 'hi', 'hi_IN', 'ar', 'ar_EG', 'ar_SA', 'fa', 'fa_IR', 'he', 'he_IL',
];

const VECTOR_CURRENCIES = ['USD', 'EUR', 'GBP', 'JPY', 'KWD', 'BHD', 'MMK', 'INR', 'CHF', 'CNY', 'THB', 'CLF', 'PTS'];

const VECTOR_VALUES = ['0', '1', '-1', '0.05', '12.34', '-1234.56', '999.99', '1000', '12345.67', '1234567.89', '-98765432.1'];

$root = dirname(__DIR__);
preg_match_all('/"([A-Z]{3})": \{code: "[A-Z]{3}", name: "[^"]*", minorUnits: (\d+)/', file_get_contents($root.'/currencies_iso.go'), $matches, PREG_SET_ORDER);
$precisions = [];
foreach ($matches as [, $code, $units]) {
    $precisions[$code] = (int) $units;
}
if (count($precisions) < 150) {
    fwrite(STDERR, "could not read currencies_iso.go\n");
    exit(1);
}

function split_affixes(string $formatted, array $digits): array
{
    $chars = preg_split('//u', $formatted, -1, PREG_SPLIT_NO_EMPTY);
    $positions = array_keys(array_filter($chars, static fn (string $c): bool => in_array($c, $digits, true)));

    return [implode('', array_slice($chars, 0, min($positions))), implode('', array_slice($chars, max($positions) + 1))];
}

/** Characters that do not trigger currency spacing: CLDR [:S:] and [:Z:]. */
function is_symbol_or_space(string $char): bool
{
    return preg_match('/^[\p{S}\p{Z}]$/u', $char) === 1;
}

function first_char(string $s): string
{
    return mb_substr($s, 0, 1);
}

function last_char(string $s): string
{
    return mb_substr($s, -1);
}

/** The same rules as renderAffix in format.go. */
function render(string $template, string $symbol, bool $prefix): string
{
    if (! str_contains($template, '¤')) {
        return $template;
    }
    $out = str_replace('¤', $symbol, $template);
    if ($prefix && str_ends_with($template, '¤') && ! is_symbol_or_space(last_char($symbol))) {
        $out .= "\u{00A0}";
    }
    if (! $prefix && str_starts_with($template, '¤') && ! is_symbol_or_space(first_char($symbol))) {
        $out = "\u{00A0}".$out;
    }

    return $out;
}

/** Turn ICU affixes for a currency back into a template by replacing its symbol with "¤". */
function template(string $affix, string $symbol): string
{
    $pos = $symbol === '' ? false : strpos($affix, $symbol);

    return $pos === false ? $affix : substr($affix, 0, $pos).'¤'.substr($affix, $pos + strlen($symbol));
}

function go_string(string $s): string
{
    $out = '"';
    foreach (preg_split('//u', $s, -1, PREG_SPLIT_NO_EMPTY) as $char) {
        $cp = mb_ord($char);
        $out .= match (true) {
            $char === '"' => '\\"',
            $char === '\\' => '\\\\',
            $cp >= 0x20 && $cp < 0x7F => $char,
            $cp <= 0xFFFF => sprintf('\\u%04X', $cp),
            default => sprintf('\\U%08X', $cp),
        };
    }

    return $out.'"';
}

$locales = [];
$vectors = [];
$overrideCount = 0;

foreach (LOCALES as $locale) {
    $formatter = new NumberFormatter($locale, NumberFormatter::CURRENCY);
    $zero = mb_ord($formatter->getSymbol(NumberFormatter::ZERO_DIGIT_SYMBOL));
    $digits = array_map(static fn (int $i): string => mb_chr($zero + $i), range(0, 9));
    $group = $formatter->getSymbol(NumberFormatter::MONETARY_GROUPING_SEPARATOR_SYMBOL);
    $primary = $formatter->getAttribute(NumberFormatter::GROUPING_USED) ? $formatter->getAttribute(NumberFormatter::GROUPING_SIZE) : 0;
    $secondary = $formatter->getAttribute(NumberFormatter::SECONDARY_GROUPING_SIZE);
    $secondary = $secondary > 0 ? $secondary : $primary;
    $minimum = 1;
    if ($primary > 0 && $group !== '') {
        for ($minimum = 1; $minimum < 4; $minimum++) {
            if (str_contains($formatter->format((int) ('1'.str_repeat('0', $primary + $minimum - 1))), $group)) {
                break;
            }
        }
    }

    $affixes = [];
    $symbols = [];
    $templates = [];
    foreach (array_keys($precisions) as $code) {
        $formatter->setTextAttribute(NumberFormatter::CURRENCY_CODE, $code);
        $formatter->setAttribute(NumberFormatter::MIN_FRACTION_DIGITS, 0);
        $formatter->setAttribute(NumberFormatter::MAX_FRACTION_DIGITS, 0);
        $symbol = $formatter->getSymbol(NumberFormatter::CURRENCY_SYMBOL);
        $pos = split_affixes($formatter->formatCurrency(1, $code), $digits);
        $neg = split_affixes($formatter->formatCurrency(-1, $code), $digits);
        $affixes[$code] = [$pos[0], $pos[1], $neg[0], $neg[1]];
        $symbols[$code] = $symbol;
        // Only symbols that never trigger currency spacing ("$", "€") show
        // the locale's pattern unchanged.
        if (is_symbol_or_space(first_char($symbol)) && is_symbol_or_space(last_char($symbol))) {
            $key = implode("\0", [template($pos[0], $symbol), template($pos[1], $symbol), template($neg[0], $symbol), template($neg[1], $symbol)]);
            $templates[$key] = ($templates[$key] ?? 0) + 1;
        }
    }
    arsort($templates);
    $template = explode("\0", array_key_first($templates));

    $localeSymbols = [];
    $overrides = [];
    foreach ($affixes as $code => $expected) {
        $symbol = $symbols[$code];
        if ($symbol !== $code) {
            $localeSymbols[$code] = $symbol;
        }
        $rebuilt = [render($template[0], $symbol, true), render($template[1], $symbol, false), render($template[2], $symbol, true), render($template[3], $symbol, false)];
        if ($rebuilt !== $expected) {
            $overrides[$code] = $expected;
            $overrideCount++;
        }
    }
    ksort($localeSymbols);

    $locales[$locale] = [
        'decimal' => $formatter->getSymbol(NumberFormatter::MONETARY_SEPARATOR_SYMBOL),
        'group' => $group,
        'primary' => $primary,
        'secondary' => $secondary,
        'minimum' => $minimum,
        'zero' => $zero,
        'template' => $template,
        'symbols' => $localeSymbols,
        'overrides' => $overrides,
    ];

    foreach (VECTOR_CURRENCIES as $code) {
        $precision = $precisions[$code] ?? 2;
        $formatter->setTextAttribute(NumberFormatter::CURRENCY_CODE, $code);
        $formatter->setAttribute(NumberFormatter::MIN_FRACTION_DIGITS, $precision);
        $formatter->setAttribute(NumberFormatter::MAX_FRACTION_DIGITS, $precision);
        foreach (VECTOR_VALUES as $value) {
            [$integer, $fraction] = array_pad(explode('.', $value, 2), 2, '');
            if (strlen($fraction) > $precision) {
                continue;
            }
            $decimal = $integer.($precision > 0 ? '.'.str_pad($fraction, $precision, '0') : '');
            $vectors[] = implode("\t", [$locale, $code, $precision, $decimal, $formatter->formatCurrency((float) $decimal, $code)]);
        }
    }
}

$go = "// Code generated by scripts/generate-formats.php from ICU ".INTL_ICU_VERSION." (CLDR data). DO NOT EDIT.\n\npackage money\n\n";
$go .= "// icuVersion is the ICU release the locale data was generated from.\nconst icuVersion = \"".INTL_ICU_VERSION."\"\n\n";
$go .= "// cldrLocales holds the currency format of every supported locale.\nvar cldrLocales = map[string]*localeFormat{\n";
foreach ($locales as $locale => $data) {
    $go .= "\t".go_string($locale).": {\n";
    $go .= "\t\tdecimal: ".go_string($data['decimal']).", group: ".go_string($data['group']).",\n";
    $go .= sprintf("\t\tprimary: %d, secondary: %d, minimum: %d, zero: 0x%04X,\n", $data['primary'], $data['secondary'], $data['minimum'], $data['zero']);
    $go .= "\t\taffixes: [4]string{".implode(', ', array_map('go_string', $data['template']))."},\n";
    if ($data['symbols'] !== []) {
        $go .= "\t\tsymbols: map[string]string{";
        $pairs = [];
        foreach ($data['symbols'] as $code => $symbol) {
            $pairs[] = go_string($code).': '.go_string($symbol);
        }
        $go .= implode(', ', $pairs)."},\n";
    }
    if ($data['overrides'] !== []) {
        $go .= "\t\toverrides: map[string][4]string{\n";
        foreach ($data['overrides'] as $code => $affixes) {
            $go .= "\t\t\t".go_string($code).': {'.implode(', ', array_map('go_string', $affixes))."},\n";
        }
        $go .= "\t\t},\n";
    }
    $go .= "\t},\n";
}
$go .= "}\n";

file_put_contents($root.'/formats_cldr.go', $go);
@mkdir($root.'/testdata');
file_put_contents($root.'/testdata/formats.tsv', "# Generated by scripts/generate-formats.php from ICU ".INTL_ICU_VERSION.": locale, currency, decimals, amount, ICU output.\n".implode("\n", $vectors)."\n");

fwrite(STDERR, sprintf("%d locales, %d overrides, %d vectors\n", count($locales), $overrideCount, count($vectors)));
