package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/asecurityteam/rolling"
	cbp "github.com/preichenberger/go-coinbasepro/v2"
)

func main() {

	sec := 10
	windowSize := 10
	windowDuration := (time.Duration(sec*windowSize) * time.Second)
	currency := "BTC"
	tslPercent := 0.05
	new := false

	// set up a log file in the samem directory
	logfile, err := os.OpenFile("tsl.log", os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		log.Fatalf("[error]   opening log file: %v\n", err)
	}
	defer logfile.Close()
	log.SetOutput(logfile)

	// set up coinbase pro client
	client := cbp.NewClient()
	client.HTTPClient = &http.Client{
		Timeout: 15 * time.Second,
	}

	// get my btc account
	btc, err := GetAccount(client, currency)
	if err != nil {
		log.Fatalf("[error]   getting btc account: %v\n", err)
	}

	// config the TSL order
	tsl := TrailingStopLoss{
		client:     client,
		account:    btc,
		tslPercent: tslPercent,
	}

	// get existing stop order if one exists and set it to the current tsl
	eo, err := GetExistingOrder(client, currency)
	if err != nil {
		log.Println("[warn]   didn't find existing orders")
		new = true
	} else {
		tsl.sellPrice, _ = strconv.ParseFloat(eo.Price, 32)
		tsl.order = eo
		log.Printf("[info]   found exiting order, price %v\n", eo.Price)
		new = false
	}

	// cancel all existing orders
	//CancelExistingOrders(client, currency)

	// set up a rolling window slice for the last 'windowSize' # of prices
	var lastPrices = rolling.NewPointPolicy(rolling.NewWindow(windowSize))

	sw := StopWatch{
		Start: time.Now(),
	}
	// loop until sell is made
	for {
		// wait to get next price
		wait(sec)

		price, err := getCurrentPrice(client)
		if err != nil {
			continue
		}
		// add the newest price to the rolling window
		lastPrices.Append(price)

		// if enough time has elapsed log information
		if sw.Elapsed()+time.Second > windowDuration {
			logInfo(&tsl, (sec * windowSize), lastPrices, price)
			sw.Reset()
		}

		// if no order is placed, place one
		// if the current price is more than the rolling max
		// update the trailing stop order to a higher value
		if new {
			tsl.ChangeSellPrice(price)
			tsl.CreateOrder()
			new = false
		} else if price > lastPrices.Reduce(rolling.Max) && price*(1-tslPercent) > tsl.sellPrice {
			fmt.Println("order")
			fmt.Println(price, lastPrices.Reduce(rolling.Max))
			tsl.UpdateOrder(price)
		}

		// if we have executed the trailing stop order
		// exit the program
		onSellLogAndExit(&tsl)
	}

}

// if the trade is executed, exit the program
func onSellLogAndExit(tsl *TrailingStopLoss) {
	if tsl.order.Status == "done" {
		log.Printf("[exe]     sell executed for %s\n", tsl.order.ExecutedValue)
		os.Exit(1)
	}
}

// get the current price of bitcoin
func getCurrentPrice(c *cbp.Client) (float64, error) {
	// get the bitcoin ticker, used to get prices
	ticker, err := c.GetTicker("BTC-USD")
	if err != nil {
		log.Fatalf("[error]   finding btc ticker: %v\n", err)
	}
	// get the current price of bitcoin
	curPrice, err := strconv.ParseFloat(ticker.Price, 32)
	if err != nil {
		log.Println("[warn]    could not get a new btc price")
	}
	return curPrice, err
}

// helper function to log stats
func logInfo(tsl *TrailingStopLoss, time int, lastPrices *rolling.PointPolicy, currentPrice float64) {
	log.Printf("[info]    %v second rolling average: %f\n", time, lastPrices.Reduce(rolling.Avg))
	log.Printf("[info]    %v second rolling max: %f\n", time, lastPrices.Reduce(rolling.Max))
	log.Printf("[info]    %v second rolling min: %f\n", time, lastPrices.Reduce(rolling.Min))
	log.Printf("[account] active stop order: %f\n", tsl.sellPrice)
	balanceValue, _ := strconv.ParseFloat(tsl.account.Balance, 32)
	balanceValue *= currentPrice
	log.Printf("[account] value: %f, balance %s\n", balanceValue, tsl.account.Balance)
}

// wait sec seconds
func wait(sec int) {
	time.Sleep(time.Duration(sec) * time.Second)
}
