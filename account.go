package main

import (
	"errors"
	"log"

	cbp "github.com/preichenberger/go-coinbasepro/v2"
)

// GetAccount finds and returns the BTC account
func GetAccount(c *cbp.Client, currency string) (cbp.Account, error) {
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

// GetExistingOrder finds existing orders with the ticker provided
func GetExistingOrder(c *cbp.Client, currency string) (cbp.Order, error) {
	var orders []cbp.Order
	cursor := c.ListOrders()

	for cursor.HasMore {
		if err := cursor.NextPage(&orders); err != nil {
			println(err.Error())
			return cbp.Order{}, errors.New("Could not find an order")
		}

		for _, o := range orders {
			if currency+"-USD" == o.ProductID {
				return o, nil
			}
		}
	}
	return cbp.Order{}, errors.New("Could not find an order")
}
