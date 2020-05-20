package main

import (
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/asecurityteam/rolling"
	cbp "github.com/preichenberger/go-coinbasepro/v2"
)

func main() {

	// set up a log file in the samem directory
	logfile, err := os.OpenFile("tsl.log", os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		log.Fatalf("[error] opening log file: %v\n", err)
	}
	defer logfile.Close()
	log.SetOutput(logfile)

	// set up coinbase pro client
	client := cbp.NewClient()
	client.HTTPClient = &http.Client{
		Timeout: 15 * time.Second,
	}

	// get the bitcoin ticker, used to get prices
	ticker, err := client.GetTicker("BTC-USD")
	if err != nil {
		log.Fatalf("[error] finding btc ticker: %v\n", err)
	}

	// get my btc account
	btc, err := GetBTCAccount(client)
	if err != nil {
		log.Fatalf("[error] getting btc account: %v\n", err)
	}

	// config the TSL order
	tsl := TrailingStopLoss{
		client:     client,
		account:    btc,
		tslPercent: 0.05,
	}

	// set up a rolling window slice for the last 10 prices
	var lastPrices = rolling.NewPointPolicy(rolling.NewWindow(10))

	start := time.Now()
	// loop until sell is made
	for {
		// wait to get next price
		time.Sleep(10 * time.Second)
		// get the current price of bitcoin
		curPrice, err := strconv.ParseFloat(ticker.Price, 32)
		if err != nil {
			log.Println("[warn] could not get a new btc price")
			continue
		}

		if time.Now().Sub(start) > (100 * time.Second) {
			log.Printf("[info] 100 second rolling average: %f\n", lastPrices.Reduce(rolling.Avg))
			log.Printf("[info] 100 second rolling max: %f\n", lastPrices.Reduce(rolling.Max))
			log.Printf("[info] 100 second rolling min: %f\n", lastPrices.Reduce(rolling.Min))
			balanceValue, _ := strconv.ParseFloat(btc.Balance, 32)
			balanceValue *= curPrice
			log.Printf("[account] value: %f, balance %s\n", balanceValue, btc.Balance)
			start = time.Now()
		}

		if curPrice > lastPrices.Reduce(rolling.Max) {
			tsl.CancelOrder()
			tsl.ChangeSellPrice(curPrice)
			tsl.CreateOrder()
			log.Printf("\n[order] placed for: %f\n\n", tsl.sellPrice)
		}
		if tsl.order.Status == "done" {
			log.Printf("[exe] sell executed for %s\n", tsl.order.ExecutedValue)
			os.Exit(1)
		}
		lastPrices.Append(curPrice)

	}

}
