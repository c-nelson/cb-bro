package main

import (
	"errors"
	"log"

	cbp "github.com/preichenberger/go-coinbasepro/v2"
)

// GetBTCAccount finds and returns the BTC account
func GetBTCAccount(c *cbp.Client) (cbp.Account, error) {
	accounts, err := c.GetAccounts()
	if err != nil {
		log.Fatalln(err)
	}
	for _, acc := range accounts {
		if acc.Currency == "BTC" {
			return acc, nil
		}
	}
	return cbp.Account{}, errors.New("Could not find BTC account")
}
