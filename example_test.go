package money_test

import (
	"errors"
	"fmt"

	money "github.com/laranex/goravel-money/v4"
)

func ExampleParse() {
	usd := money.MustCurrency("USD")

	price, _ := money.Parse("1,234.50", usd)
	fmt.Println(price.Amount(), price.Decimal())

	_, err := money.Parse("1.234", usd)
	fmt.Println(errors.Is(err, money.ErrTooManyDecimals))

	rounded, _ := money.Parse("1.235", usd, money.HalfUp)
	fmt.Println(rounded.Decimal())
	// Output:
	// 123450 1234.50
	// true
	// 1.24
}

func ExampleMoney_Plus() {
	usd := money.MustCurrency("USD")
	price := money.MustParse("19.99", usd)

	total, _ := price.Plus(money.MustParse("5.00", usd), "2.50")
	fmt.Println(total.Decimal())

	_, err := price.Plus(money.MustParse("1", money.MustCurrency("EUR")))
	fmt.Println(err)
	// Output:
	// 27.49
	// money: cannot combine USD 19.99 with EUR 1.00: the amounts are in different currencies; convert one of them first
}

func ExampleMoney_Times() {
	m := money.MustParse("0.05", money.MustCurrency("USD"))

	halfUp, _ := m.Times("0.5")
	floor, _ := m.Times("0.5", money.Floor)
	fmt.Println(halfUp.Decimal(), floor.Decimal())
	// Output: 0.03 0.02
}

func ExampleMoney_AddPercent() {
	price := money.MustParse("1,234.50", money.MustCurrency("USD"))

	withTax, _ := price.AddPercent("8.875")
	discounted, _ := price.SubtractPercent(15)
	fmt.Println(withTax.Decimal(), discounted.Decimal())
	// Output: 1344.06 1049.32
}

func ExampleMoney_Split() {
	parts, _ := money.MustParse("100", money.MustCurrency("USD")).Split(3)
	fmt.Println(parts[0].Decimal(), parts[1].Decimal(), parts[2].Decimal())
	// Output: 33.34 33.33 33.33
}

func ExampleAllocateMap() {
	total := money.MustParse("100", money.MustCurrency("USD"))

	shares, _ := money.AllocateMap(total, map[string]int{"owner": 70, "agent": 20, "platform": 10})
	fmt.Println(shares["owner"].Decimal(), shares["agent"].Decimal(), shares["platform"].Decimal())
	// Output: 70.00 20.00 10.00
}

func ExampleMoney_Format() {
	eur := money.MustParse("1234.5", money.MustCurrency("EUR"))

	fmt.Println(eur.Format("en"))
	fmt.Println(eur.Format("de_DE") == "1.234,50 €")
	fmt.Println(eur.String())
	// Output:
	// €1,234.50
	// true
	// EUR 1234.50
}

func ExampleSum() {
	usd := money.MustCurrency("USD")
	prices := []money.Money{money.MustParse("10", usd), money.MustParse("2.50", usd), money.MustParse("7.51", usd)}

	sum, _ := money.Sum(prices)
	avg, _ := money.Avg(prices)
	most, _ := money.Max(prices)
	fmt.Println(sum.Decimal(), avg.Decimal(), most.Decimal())
	// Output: 20.01 6.67 10.00
}
