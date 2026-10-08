// Package facades is the facade-style accessor for goravel-money.
package facades

import (
	money "github.com/laranex/goravel-money/v4"
)

// Money returns the application's *money.Manager. It panics when
// &money.ServiceProvider{} is not registered or money.default_currency is not
// an ISO 4217 code, like a missing facade in Goravel.
func Money() *money.Manager {
	manager, err := money.Registered()
	if err != nil {
		panic(err)
	}

	return manager
}
