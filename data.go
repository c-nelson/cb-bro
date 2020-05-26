package main

import (
	"fmt"
	"log"
	"strconv"
	"time"

	"github.com/asecurityteam/rolling"
	cbp "github.com/preichenberger/go-coinbasepro/v2"
)

// GetCurrentPrice for a currency
func GetCurrentPrice(c *cbp.Client, currency string) (float64, error) {
	// get the currency ticker, used to get prices
	ticker, err := c.GetTicker(fmt.Sprintf("%s-USD", currency))
	if err != nil {
		log.Printf("[error]   finding %s ticker: %v\n", currency, err)
	}
	// get the current price of bitcoin
	curPrice, err := strconv.ParseFloat(ticker.Price, 32)
	if err != nil {
		log.Printf("[warn]    could not get a new %s price\n", currency)
	}
	return curPrice, err
}

// LogRollingInfo given a rolling window
func LogRollingInfo(windowDuration time.Duration, window *rolling.PointPolicy) {
	log.Printf("[info]    %v second rolling average: %f\n", windowDuration, window.Reduce(rolling.Avg))
	log.Printf("[info]    %v second rolling max: %f\n", windowDuration, window.Reduce(rolling.Max))
	log.Printf("[info]    %v second rolling min: %f\n", windowDuration, window.Reduce(rolling.Min))

}

// LogAccountBalance and value
func LogAccountBalance(acc cbp.Account, currentPrice float64) {
	balanceValue, _ := strconv.ParseFloat(acc.Balance, 32)
	balanceValue *= currentPrice
	log.Printf("[account] value: %f, balance %s\n", balanceValue, acc.Balance)
}
