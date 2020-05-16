package main

import (
	"fmt"
	"net/http"
	"time"

	cbp "github.com/preichenberger/go-coinbasepro/v2"
	"github.com/shopspring/decimal"
)

func main() {

	client := cbp.NewClient()
	client.HTTPClient = &http.Client{
		Timeout: 15 * time.Second,
	}

	book, err := client.GetBook("BTC-USD", 1)
	if err != nil {
		fmt.Println(err)
	}

	lastPrice, err := decimal.NewFromString(book.Bids[0].Price)
	if err != nil {
		fmt.Println(err)
	}

	fmt.Println(lastPrice)

	accounts, err := client.GetAccounts()
	if err != nil {
		fmt.Println(err)
	}

	var btcAcc cbp.Account
	for _, acc := range accounts {
		if acc.Currency == "BTC" {
			btcAcc = acc
		}
	}
	btcBalance, _ := decimal.NewFromString(btcAcc.Balance)

	fmt.Println(btcAcc.Currency, btcBalance, (btcBalance.Mul(lastPrice)))

	// app := &cli.App{
	// 	Name:  "cbbro",
	// 	Usage: "coinbase pro CLI with some extra funcationality",
	// 	Action: func(c *cli.Context) error {
	// 		fmt.Println("boom! I say!")
	// 		return nil
	// 	},
	// }

	// err = app.Run(os.Args)
	// if err != nil {
	// 	log.Fatal(err)
	// }

}
