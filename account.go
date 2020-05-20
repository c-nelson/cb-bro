package main

import (
	"errors"
	"log"

	cbp "github.com/preichenberger/go-coinbasepro/v2"
)

// GetBTCAccount finds and returns the BTC account
func GetBTCAccount(c *cbp.Client, currency string) (cbp.Account, error) {
	accounts, err := c.GetAccounts()
	if err != nil {
		log.Fatalln("[error]   couldn't access accounts")
	}
	for _, acc := range accounts {
		if acc.Currency == currency {
			return acc, nil
		}
	}
	return cbp.Account{}, errors.New("Could not find account")
}

// CancelExistingOrders is used to cancel orders before the program starts its loop
func CancelExistingOrders(c *cbp.Client, currency string) {
	o := cbp.CancelAllOrdersParams{
		ProductID: currency + "-USD",
	}
	_, err := c.CancelAllOrders(o)
	if err != nil {
		log.Fatalln("[error]   canceling existing orders")
	}
}
