package money

import (
	"testing"

	_ "github.com/ncruces/go-sqlite3/embed"
	"github.com/ncruces/go-sqlite3/gormlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/laranex/goravel-money/v4/iso"
)

// product mirrors laravel-money's test fixture: price uses the default
// currency, cost and deposit fix theirs.
type product struct {
	ID      uint
	Price   Column[Default]
	Cost    Column[iso.MMK]
	Deposit Column[iso.EUR]
}

// TestColumnWithGorm round-trips Column through GORM (the ORM under Goravel's
// facades.Orm()) on an in-memory SQLite database.
func TestColumnWithGorm(t *testing.T) {
	useDefaultCurrency(t, "USD")

	db, err := gorm.Open(gormlite.Open("file::memory:"), &gorm.Config{Logger: logger.Discard})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&product{}))

	created := product{
		Price:   NewColumn[Default](New(1050, MustCurrency("USD"))),
		Cost:    NewColumn[iso.MMK](New(250000, MustCurrency("MMK"))),
		Deposit: NewColumn[iso.EUR](New(999, MustCurrency("EUR"))),
	}
	require.NoError(t, db.Create(&created).Error)

	var raw struct{ Price, Cost, Deposit int64 }
	require.NoError(t, db.Raw("SELECT price, cost, deposit FROM products WHERE id = ?", created.ID).Scan(&raw).Error)
	assert.Equal(t, int64(1050), raw.Price, "stored as minor units")
	assert.Equal(t, int64(250000), raw.Cost)
	assert.Equal(t, int64(999), raw.Deposit)

	var fresh product
	require.NoError(t, db.First(&fresh, created.ID).Error)
	assert.Equal(t, created.Price, fresh.Price)
	assert.Equal(t, "MMK", fresh.Cost.Money.Currency().Code())
	assert.Equal(t, "10.50", fresh.Price.Money.Decimal())
	assert.Equal(t, "9.99", fresh.Deposit.Money.Decimal())

	nulls := product{}
	require.NoError(t, db.Create(&nulls).Error)
	var freshNulls product
	require.NoError(t, db.First(&freshNulls, nulls.ID).Error)
	assert.False(t, freshNulls.Price.Valid)
	assert.False(t, freshNulls.Cost.Valid)

	mismatch := product{Cost: NewColumn[iso.MMK](New(1, MustCurrency("USD")))}
	assert.ErrorIs(t, db.Create(&mismatch).Error, ErrCurrencyMismatch)
}
