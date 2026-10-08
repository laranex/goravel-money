package facades

import (
	"testing"

	mocksfoundation "github.com/goravel/framework/mocks/foundation"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	money "github.com/laranex/goravel-money/v4"
)

func TestMoneyPanicsWithoutTheServiceProvider(t *testing.T) {
	assert.PanicsWithError(t, "goravel-money: register &money.ServiceProvider{} in bootstrap/providers.go", func() { Money() })
}

func TestMoneyResolvesTheBoundManager(t *testing.T) {
	manager, err := money.NewManager("MMK")
	require.NoError(t, err)

	app := mocksfoundation.NewApplication(t)
	app.EXPECT().Singleton(money.Binding, mock.Anything).Once()
	app.EXPECT().Make(money.Binding).Return(manager, nil).Once()
	(&money.ServiceProvider{}).Register(app)

	assert.Same(t, manager, Money())
	assert.Equal(t, "MMK", manager.DefaultCurrency().Code())
}
