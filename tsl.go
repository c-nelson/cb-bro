package main

import (
	"fmt"
	"log"
	"strconv"
	"time"

	"github.com/asecurityteam/rolling"
	cbp "github.com/preichenberger/go-coinbasepro/v2"
)

// TrailingStopLoss holds information about a trailing stop loss order
type TrailingStopLoss struct {
	sellPrice      float64              // The price the TSL will sell (1 - tslPercent) * Market Price
	tslPercent     float64              // Percentage to set the TSL, ex 0.05 = 5%
	updated        time.Time            // Used to track when events occur
	order          cbp.Order            // The active TSL order
	client         *cbp.Client          // The client used to communicate with Coinbase Pro
	account        cbp.Account          // The "account" or currency we are trading on
	waitTime       int                  // The time to wait in between getting new prices, in seconds
	currency       string               // The currency we are trading, ex "BTC"
	windowSize     int                  // The size of the rolling window of prices
	windowDuration time.Duration        // The seconds passed in 1 window cycle
	window         *rolling.PointPolicy // The rolling window to used to update orders, ex if the window size is 10 the TSL will only update if the maximum price in the window is greater than sellPrice * (1 - tslPercent)
	stopWatch      StopWatch            // Used to only log non critical information periodically
	currentPrice   float64              // Used to store the current market price
}

// GetOrder for active TSL
func (tsl *TrailingStopLoss) GetOrder() cbp.Order {
	return tsl.order
}

// ChangeSellPrice calculates the sell price based on the TSL percent
func (tsl *TrailingStopLoss) ChangeSellPrice(market float64) {
	tsl.sellPrice = market * (1 - tsl.tslPercent)
}

// UpdateTime is used to track the time a change to the TSL was last made
func (tsl *TrailingStopLoss) UpdateTime() {
	tsl.updated = time.Now()
}

// SetAccount sets the account the TSL will take place on
func (tsl *TrailingStopLoss) SetAccount(a cbp.Account) {
	tsl.account = a
}

// CreateOrder creates an order based on the values of the TSL and saves the order to tsl.order
func (tsl *TrailingStopLoss) CreateOrder() {
	tsl.order = cbp.Order{
		Price:       fmt.Sprintf("%.2f", tsl.sellPrice),
		Size:        tsl.account.Balance,
		Side:        "sell",
		Stop:        "loss",
		StopPrice:   fmt.Sprintf("%.2f", tsl.sellPrice),
		TimeInForce: "GTC",
		ProductID:   fmt.Sprintf("%s-USD", tsl.account.Currency),
	}
	savedOrder, err := tsl.client.CreateOrder(&tsl.order)
	if err != nil {
		log.Fatalln(err.Error())
	}
	tsl.order = savedOrder
	log.Printf("[order]   placed for: %f\n", tsl.sellPrice)
	fmt.Printf("[order]   placed for: %f\n", tsl.sellPrice)
	tsl.UpdateTime()
}

// CancelOrder cancels the TSL
func (tsl *TrailingStopLoss) CancelOrder() {
	if tsl.order == (cbp.Order{}) {
		log.Println("[warn]    no order to cancel")
	} else {
		err := tsl.client.CancelOrder(tsl.order.ID)
		if err != nil {
			log.Println("[warn]    could not cancel order")
		}
		tsl.order = cbp.Order{}
		tsl.UpdateTime()
	}
}

// UpdateOrder with newest pricing information
func (tsl *TrailingStopLoss) UpdateOrder(newPrice float64) {
	if tsl.order != (cbp.Order{}) {
		tsl.CancelOrder()
	}
	tsl.ChangeSellPrice(newPrice)
	tsl.CreateOrder()
}

// Run is an infinite loop used to update the TSL
func (tsl *TrailingStopLoss) Run() {
	tsl.init()
	// loop until sell is made
	for !tsl.isSellMade() {
		// wait to get next price
		Wait(tsl.waitTime)

		var err error
		tsl.currentPrice, err = GetCurrentPrice(tsl.client, tsl.currency)
		if err != nil {
			continue
		}
		// add the newest price to the rolling window
		tsl.window.Append(tsl.currentPrice)
		tsl.logAllInfoAtInterval()
		// if the current price is more than the rolling max
		// update the trailing stop order to a higher value
		if tsl.isUpdateOrderConditionMet() {
			tsl.UpdateOrder(tsl.currentPrice)
		}
	}
}

func (tsl *TrailingStopLoss) init() {
	tsl.getCryptoAccount()
	tsl.checkForExistingOrder()
	tsl.initWindowDuration()
	tsl.initRollingWindow()
	tsl.stopWatch = StopWatch{
		Start: time.Now(),
	}

}

func (tsl *TrailingStopLoss) isUpdateOrderConditionMet() bool {
	return tsl.window.Reduce(rolling.Max)*(1-tsl.tslPercent) > tsl.sellPrice
}

func (tsl *TrailingStopLoss) getCryptoAccount() {
	// get my account
	cur, err := GetAccount(tsl.client, tsl.currency)
	if err != nil {
		log.Fatalf("[error]   getting btc account: %v\n", err)
	}
	tsl.account = cur
}

func (tsl *TrailingStopLoss) checkForExistingOrder() {
	// get existing stop order if one exists and set it to the current tsl
	eo, err := GetExistingOrder(tsl.client, tsl.currency)
	if err != nil {
		log.Println("[warn]   didn't find existing orders")
		price, _ := GetCurrentPrice(tsl.client, tsl.currency)
		tsl.ChangeSellPrice(price)
		tsl.CreateOrder()
	} else {
		tsl.sellPrice, _ = strconv.ParseFloat(eo.Price, 32)
		tsl.order = eo
		log.Printf("[info]   found exiting order, price %v\n", eo.Price)
	}
}

func (tsl *TrailingStopLoss) initRollingWindow() {
	tsl.window = rolling.NewPointPolicy(rolling.NewWindow(tsl.windowSize))
}

func (tsl *TrailingStopLoss) initWindowDuration() {
	tsl.windowDuration = (time.Duration(tsl.waitTime*tsl.windowSize) * time.Second)
}

func (tsl *TrailingStopLoss) logActiveStopOrder() {
	log.Printf("[account] active stop order: %f\n", tsl.sellPrice)
}

// LogAllInfoAtInterval uses the tsl stopwatch to log rolling info, acccount balance, and active orders
func (tsl *TrailingStopLoss) logAllInfoAtInterval() {
	// if enough time has elapsed log information
	if tsl.stopWatch.Elapsed()+time.Second > tsl.windowDuration {
		LogRollingInfo(tsl.windowDuration, tsl.window)
		LogAccountBalance(tsl.account, tsl.currentPrice)
		tsl.logActiveStopOrder()
		tsl.stopWatch.Reset()
	}
}

func (tsl *TrailingStopLoss) isSellMade() bool {
	return tsl.order.Status == "done"
}
