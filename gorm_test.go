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

// product mirrors laravel-money's tests/Fixtures/Product.php: price uses the
// default currency, cost/deposit/yen/points fix theirs, fee and kwd are
// DECIMAL columns, and balance/total read their currency from Currency.
type product struct {
	ID       uint
	Price    Column[Default]
	Cost     Column[iso.MMK]
	Deposit  Column[iso.EUR]
	Yen      Column[iso.JPY]
	Points   Column[PTS]
	Fee      DecimalColumn[iso.USD]
	Kwd      DecimalColumn[iso.KWD]
	Currency string
	Balance  AmountColumn
	Total    DecimalAmountColumn
}

// openDB returns GORM (the ORM behind Goravel's facades.Orm()) on an
// in-memory SQLite database with the products table.
func openDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(gormlite.Open("file::memory:"), &gorm.Config{Logger: logger.Discard})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&product{}))
	t.Cleanup(func() {
		if sqlDB, err := db.DB(); err == nil {
			_ = sqlDB.Close()
		}
	})

	return db
}

func fresh(t *testing.T, db *gorm.DB, id uint) product {
	t.Helper()
	var p product
	require.NoError(t, db.First(&p, id).Error)

	return p
}

func TestIntegerColumnsWithGorm(t *testing.T) {
	useConfig(t, laravelConfig())
	db := openDB(t)

	created := product{
		Price:   NewColumn[Default](of(t, "10.50", "")),
		Cost:    NewColumn[iso.MMK](of(t, "2500", "MMK")),
		Deposit: NewColumn[iso.EUR](of(t, "9.99", "EUR")),
		Yen:     NewColumn[iso.JPY](of(t, "1500", "JPY")),
		Points:  NewColumn[PTS](of(t, "300", "PTS")),
	}
	require.NoError(t, db.Create(&created).Error)

	var raw struct{ Price, Cost, Deposit, Yen, Points int64 }
	require.NoError(t, db.Raw("SELECT price, cost, deposit, yen, points FROM products WHERE id = ?", created.ID).Scan(&raw).Error)
	assert.Equal(t, struct{ Price, Cost, Deposit, Yen, Points int64 }{1050, 250000, 999, 1500, 300}, raw, "stored as minor units")

	got := fresh(t, db, created.ID)
	assert.Equal(t, created.Price, got.Price)
	assert.Equal(t, "2500.00", got.Cost.Money.Decimal())
	assert.Equal(t, "EUR", got.Deposit.Money.Currency().Code())
	assert.Equal(t, "1500", got.Yen.Money.Decimal())
	assert.Equal(t, "PTS 300", got.Points.Money.String())
	assert.False(t, got.Fee.Valid)
	assert.False(t, got.Balance.Valid)
}

func TestIntegerColumnsRoundTripArithmetic(t *testing.T) {
	useConfig(t, laravelConfig())
	db := openDB(t)

	created := product{Price: NewColumn[Default](of(t, "10", ""))}
	require.NoError(t, db.Create(&created).Error)

	price, err := created.Price.Money.Plus("2.50")
	require.NoError(t, err)
	price, err = price.AddPercent(10)
	require.NoError(t, err)
	created.Price = NewColumn[Default](price)
	require.NoError(t, db.Save(&created).Error)

	assert.Equal(t, "13.75", fresh(t, db, created.ID).Price.Money.Decimal())
}

// ledger stores amounts beyond int64: SQLite turns such integers into REAL in
// a BIGINT column, so they need an exact column type (NUMERIC(n, 0) on
// PostgreSQL and MySQL, TEXT on SQLite).
type ledger struct {
	ID    uint
	Total Column[iso.USD] `gorm:"type:text"`
}

func TestIntegerColumnsBeyondInt64(t *testing.T) {
	useConfig(t, laravelConfig())
	db := openDB(t)
	require.NoError(t, db.AutoMigrate(&ledger{}))

	huge := of(t, "123456789012345678901234567890.12", "USD")
	created := ledger{Total: NewColumn[iso.USD](huge)}
	require.NoError(t, db.Create(&created).Error)
	var got ledger
	require.NoError(t, db.First(&got, created.ID).Error)
	assert.True(t, equal(t, huge, got.Total.Money))

	lossy := product{Price: NewColumn[Default](huge)}
	require.NoError(t, db.Create(&lossy).Error)
	var p product
	err := db.First(&p, lossy.ID).Error
	assert.ErrorIs(t, err, ErrInvalidStoredAmount, "a BIGINT column on SQLite returns REAL, which is refused instead of losing digits")
}

func TestColumnsKeepNull(t *testing.T) {
	useConfig(t, laravelConfig())
	db := openDB(t)

	created := product{}
	require.NoError(t, db.Create(&created).Error)

	got := fresh(t, db, created.ID)
	assert.False(t, got.Price.Valid)
	assert.False(t, got.Fee.Valid)
	assert.False(t, got.Balance.Valid)
	assert.False(t, got.Total.Valid)
	assert.Equal(t, "", got.Currency)
}

func TestColumnsRejectMoneyInAnotherCurrency(t *testing.T) {
	useConfig(t, laravelConfig())
	db := openDB(t)

	err := db.Create(&product{Cost: NewColumn[iso.MMK](of(t, "1", "USD"))}).Error
	require.ErrorIs(t, err, ErrCurrencyMismatch)
	assert.ErrorContains(t, err, "money: the column stores MMK amounts, but USD 1.00 was given; convert it to MMK first")

	assert.ErrorIs(t, db.Create(&product{Fee: NewDecimalColumn[iso.USD](of(t, "1", "EUR"))}).Error, ErrCurrencyMismatch)
}

func TestColumnsFollowTheConfiguredDefaultCurrency(t *testing.T) {
	cfg := laravelConfig()
	cfg.DefaultCurrency = "MMK"
	useConfig(t, cfg)
	db := openDB(t)

	created := product{Price: NewColumn[Default](of(t, "5000", ""))}
	require.NoError(t, db.Create(&created).Error)
	assert.Equal(t, "MMK 5000.00", fresh(t, db, created.ID).Price.Money.String())

	assert.ErrorIs(t, db.Create(&product{Price: NewColumn[Default](of(t, "1", "USD"))}).Error, ErrCurrencyMismatch)
}

func TestDecimalColumnsWithGorm(t *testing.T) {
	useConfig(t, laravelConfig())
	db := openDB(t)

	created := product{
		Fee: NewDecimalColumn[iso.USD](of(t, "12.34", "USD")),
		Kwd: NewDecimalColumn[iso.KWD](of(t, "1.005", "KWD")),
	}
	require.NoError(t, db.Create(&created).Error)

	got := fresh(t, db, created.ID)
	assert.Equal(t, "1234", got.Fee.Money.Amount())
	assert.Equal(t, "USD", got.Fee.Money.Currency().Code())
	assert.Equal(t, "1.005", got.Kwd.Money.Decimal())

	whole := product{Fee: NewDecimalColumn[iso.USD](of(t, "12.00", "USD"))}
	require.NoError(t, db.Create(&whole).Error)
	assert.Equal(t, "12.00", fresh(t, db, whole.ID).Fee.Money.Decimal(), "SQLite returns 12 as an integer")
}

func TestDecimalColumnsReadStrictly(t *testing.T) {
	useConfig(t, laravelConfig())
	db := openDB(t)

	require.NoError(t, db.Exec("INSERT INTO products (fee) VALUES ('12.345')").Error)
	var p product
	err := db.Last(&p).Error
	assert.ErrorIs(t, err, ErrInvalidStoredAmount, "SQLite returns REAL 12.345, which is not an exact USD amount")
}

func TestCurrencyColumnWithGorm(t *testing.T) {
	useConfig(t, laravelConfig())
	db := openDB(t)

	var created product
	require.NoError(t, created.Balance.Set(of(t, "1500", "JPY"), &created.Currency))
	assert.Equal(t, "JPY", created.Currency, "filled because the column was empty")
	require.NoError(t, db.Create(&created).Error)

	got := fresh(t, db, created.ID)
	balance, ok, err := got.Balance.Money(got.Currency)
	require.NoError(t, err)
	require.True(t, ok)
	assert.Equal(t, "JPY 1500", balance.String())

	var raw struct{ Balance int64 }
	require.NoError(t, db.Raw("SELECT balance FROM products WHERE id = ?", created.ID).Scan(&raw).Error)
	assert.Equal(t, int64(1500), raw.Balance)
}

func TestCurrencyColumnReadsEveryRowsCurrency(t *testing.T) {
	useConfig(t, laravelConfig())
	db := openDB(t)

	usd := product{Currency: "USD"}
	require.NoError(t, usd.Balance.Set(of(t, "10.50", "USD"), &usd.Currency))
	kwd := product{Currency: "KWD"}
	require.NoError(t, kwd.Balance.Set(of(t, "10.5", "KWD"), &kwd.Currency))
	require.NoError(t, db.Create(&usd).Error)
	require.NoError(t, db.Create(&kwd).Error)

	var rows []product
	require.NoError(t, db.Order("id").Find(&rows).Error)
	require.Len(t, rows, 2)
	first, _, err := rows[0].Balance.Money(rows[0].Currency)
	require.NoError(t, err)
	second, _, err := rows[1].Balance.Money(rows[1].Currency)
	require.NoError(t, err)
	assert.Equal(t, "1050", first.Amount())
	assert.Equal(t, "10500", second.Amount())
	assert.Equal(t, "10.500", second.Decimal())
}

func TestCurrencyColumnMismatchAndDefault(t *testing.T) {
	useConfig(t, laravelConfig())

	p := product{Currency: "MMK"}
	err := p.Balance.Set(of(t, "1", "USD"), &p.Currency)
	require.ErrorIs(t, err, ErrCurrencyMismatch)
	assert.EqualError(t, err, "money: the currency column holds MMK, but USD 1.00 was given; convert the amount, or change the currency column first")

	db := openDB(t)
	require.NoError(t, db.Exec("INSERT INTO products (balance) VALUES (200)").Error)
	var got product
	require.NoError(t, db.Last(&got).Error)
	balance, _, err := got.Balance.Money(got.Currency)
	require.NoError(t, err)
	assert.Equal(t, "USD 2.00", balance.String(), "an empty currency column uses the default currency")
}

func TestDecimalCurrencyColumnWithGorm(t *testing.T) {
	useConfig(t, laravelConfig())
	db := openDB(t)

	var created product
	require.NoError(t, created.Total.Set(of(t, "3.125", "KWD"), &created.Currency))
	require.NoError(t, db.Create(&created).Error)
	assert.Equal(t, "KWD", created.Currency)

	got := fresh(t, db, created.ID)
	total, ok, err := got.Total.Money(got.Currency)
	require.NoError(t, err)
	require.True(t, ok)
	assert.Equal(t, "3.125", total.Decimal())
}

func TestCurrencyColumnIsSharedBetweenAmounts(t *testing.T) {
	useConfig(t, laravelConfig())
	db := openDB(t)

	created := product{Currency: "EUR"}
	require.NoError(t, created.Balance.Set(of(t, "5", "EUR"), &created.Currency))
	require.NoError(t, created.Total.Set(of(t, "7.25", "EUR"), &created.Currency))
	require.NoError(t, db.Create(&created).Error)

	got := fresh(t, db, created.ID)
	balance, _, err := got.Balance.Money(got.Currency)
	require.NoError(t, err)
	total, _, err := got.Total.Money(got.Currency)
	require.NoError(t, err)
	assert.Equal(t, "EUR 5.00", balance.String())
	assert.Equal(t, "EUR 7.25", total.String())
}
