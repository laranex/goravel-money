package money

// FallbackCurrency is used when the money.default_currency config value and
// the MONEY_CURRENCY environment variable are both unset.
const FallbackCurrency = "USD"

// Manager creates, parses and formats Money with a configured default
// currency. The service provider binds one into the container; resolve it with
// facades.Money(). It is safe for concurrent use.
type Manager struct {
	defaultCurrency Currency
}

// NewManager returns a Manager whose default currency is the given ISO 4217
// code; an empty code uses FallbackCurrency.
func NewManager(defaultCurrency string) (*Manager, error) {
	if defaultCurrency == "" {
		defaultCurrency = FallbackCurrency
	}
	currency, err := LookupCurrency(defaultCurrency)
	if err != nil {
		return nil, err
	}

	return &Manager{defaultCurrency: currency}, nil
}

// DefaultCurrency returns the currency used when none is given.
func (m *Manager) DefaultCurrency() Currency {
	return m.defaultCurrency
}

// Currency resolves an ISO 4217 code; an empty code returns the default currency.
func (m *Manager) Currency(code string) (Currency, error) {
	if code == "" {
		return m.defaultCurrency, nil
	}

	return LookupCurrency(code)
}

// Make returns Money for an amount in minor units (cents): Make(1050, "") is
// 10.50 in the default currency, Make(1050, "MMK") is 10.50 MMK.
func (m *Manager) Make(amount int64, code string) (Money, error) {
	currency, err := m.Currency(code)
	if err != nil {
		return Money{}, err
	}

	return New(amount, currency), nil
}

// Parse converts a decimal string such as "10.50" into Money in the given
// currency (empty code: the default currency). See the package-level Parse.
func (m *Manager) Parse(decimal, code string) (Money, error) {
	currency, err := m.Currency(code)
	if err != nil {
		return Money{}, err
	}

	return Parse(decimal, currency)
}

// Format returns money as a plain decimal string such as "10.50".
func (m *Manager) Format(money Money) string {
	return money.Decimal()
}
