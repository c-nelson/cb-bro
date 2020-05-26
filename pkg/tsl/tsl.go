package tsl

import (
	"fmt"
	"log"
	"strconv"
	"time"

	"github.com/asecurityteam/rolling"
	"github.com/c-nelson/cbbro/pkg/account"
	"github.com/c-nelson/cbbro/pkg/data"
	"github.com/c-nelson/cbbro/pkg/stopwatch"
	"github.com/c-nelson/cbbro/pkg/wait"
	cbp "github.com/preichenberger/go-coinbasepro/v2"
)

// TrailingStopLoss holds information about a trailing stop loss order
type TrailingStopLoss struct {
	SellPrice      float64              // The price the TSL will sell (1 - tslPercent) * Market Price
	TSLPercent     float64              // Percentage to set the TSL, ex 0.05 = 5%
	Updated        time.Time            // Used to track when events occur
	Order          cbp.Order            // The active TSL order
	Client         *cbp.Client          // The client used to communicate with Coinbase Pro
	Account        cbp.Account          // The "account" or currency we are trading on
	WaitTime       int                  // The time to wait in between getting new prices, in seconds
	Currency       string               // The currency we are trading, ex "BTC"
	WindowSize     int                  // The size of the rolling window of prices
	WindowDuration time.Duration        // The seconds passed in 1 window cycle
	Window         *rolling.PointPolicy // The rolling window to used to update orders, ex if the window size is 10 the TSL will only update if the maximum price in the window is greater than sellPrice * (1 - tslPercent)
	StopWatch      stopwatch.StopWatch  // Used to only log non critical information periodically
	CurrentPrice   float64              // Used to store the current market price
}

// GetOrder for active TSL
func (tsl *TrailingStopLoss) GetOrder() cbp.Order {
	return tsl.Order
}

// ChangeSellPrice calculates the sell price based on the TSL percent
func (tsl *TrailingStopLoss) ChangeSellPrice(market float64) {
	tsl.SellPrice = market * (1 - tsl.TSLPercent)
}

// UpdateTime is used to track the time a change to the TSL was last made
func (tsl *TrailingStopLoss) UpdateTime() {
	tsl.Updated = time.Now()
}

// SetAccount sets the account the TSL will take place on
func (tsl *TrailingStopLoss) SetAccount(a cbp.Account) {
	tsl.Account = a
}

// CreateOrder creates an order based on the values of the TSL and saves the order to tsl.order
func (tsl *TrailingStopLoss) CreateOrder() {
	tsl.Order = cbp.Order{
		Price:       fmt.Sprintf("%.2f", tsl.SellPrice),
		Size:        tsl.Account.Balance,
		Side:        "sell",
		Stop:        "loss",
		StopPrice:   fmt.Sprintf("%.2f", tsl.SellPrice),
		TimeInForce: "GTC",
		ProductID:   fmt.Sprintf("%s-USD", tsl.Account.Currency),
	}
	savedOrder, err := tsl.Client.CreateOrder(&tsl.Order)
	if err != nil {
		log.Fatalln(err.Error())
	}
	tsl.Order = savedOrder
	log.Printf("[order]   placed for: %f\n", tsl.SellPrice)
	fmt.Printf("[order]   placed for: %f\n", tsl.SellPrice)
	tsl.UpdateTime()
}

// CancelOrder cancels the TSL
func (tsl *TrailingStopLoss) CancelOrder() {
	if tsl.Order == (cbp.Order{}) {
		log.Println("[warn]    no order to cancel")
	} else {
		err := tsl.Client.CancelOrder(tsl.Order.ID)
		if err != nil {
			log.Println("[warn]    could not cancel order")
		}
		tsl.Order = cbp.Order{}
		tsl.UpdateTime()
	}
}

// UpdateOrder with newest pricing information
func (tsl *TrailingStopLoss) UpdateOrder(newPrice float64) {
	if tsl.Order != (cbp.Order{}) {
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
		wait.Wait(tsl.WaitTime)

		var err error
		tsl.CurrentPrice, err = data.GetCurrentPrice(tsl.Client, tsl.Currency)
		if err != nil {
			continue
		}
		// add the newest price to the rolling window
		tsl.Window.Append(tsl.CurrentPrice)
		tsl.logAllInfoAtInterval()
		// if the current price is more than the rolling max
		// update the trailing stop order to a higher value
		if tsl.isUpdateOrderConditionMet() {
			tsl.UpdateOrder(tsl.CurrentPrice)
		}
	}
}

func (tsl *TrailingStopLoss) init() {
	tsl.getCryptoAccount()
	tsl.checkForExistingOrder()
	tsl.initWindowDuration()
	tsl.initRollingWindow()
	tsl.StopWatch = stopwatch.StopWatch{
		Start: time.Now(),
	}

}

func (tsl *TrailingStopLoss) isUpdateOrderConditionMet() bool {
	return tsl.Window.Reduce(rolling.Max)*(1-tsl.TSLPercent) > tsl.SellPrice
}

func (tsl *TrailingStopLoss) getCryptoAccount() {
	// get my account
	cur, err := account.GetAccount(tsl.Client, tsl.Currency)
	if err != nil {
		log.Fatalf("[error]   getting btc account: %v\n", err)
	}
	tsl.Account = cur
}

func (tsl *TrailingStopLoss) checkForExistingOrder() {
	// get existing stop order if one exists and set it to the current tsl
	eo, err := account.GetExistingOrder(tsl.Client, tsl.Currency)
	if err != nil {
		log.Println("[warn]   didn't find existing orders")
		price, _ := data.GetCurrentPrice(tsl.Client, tsl.Currency)
		tsl.ChangeSellPrice(price)
		tsl.CreateOrder()
	} else {
		tsl.SellPrice, _ = strconv.ParseFloat(eo.Price, 32)
		tsl.Order = eo
		log.Printf("[info]   found exiting order, price %v\n", eo.Price)
	}
}

func (tsl *TrailingStopLoss) initRollingWindow() {
	tsl.Window = rolling.NewPointPolicy(rolling.NewWindow(tsl.WindowSize))
}

func (tsl *TrailingStopLoss) initWindowDuration() {
	tsl.WindowDuration = (time.Duration(tsl.WaitTime*tsl.WindowSize) * time.Second)
}

func (tsl *TrailingStopLoss) logActiveStopOrder() {
	log.Printf("[account] active stop order: %f\n", tsl.SellPrice)
}

// LogAllInfoAtInterval uses the tsl stopwatch to log rolling info, acccount balance, and active orders
func (tsl *TrailingStopLoss) logAllInfoAtInterval() {
	// if enough time has elapsed log information
	if tsl.StopWatch.Elapsed()+time.Second > tsl.WindowDuration {
		data.LogRollingInfo(tsl.WindowDuration, tsl.Window)
		data.LogAccountBalance(tsl.Account, tsl.CurrentPrice)
		tsl.logActiveStopOrder()
		tsl.StopWatch.Reset()
	}
}

func (tsl *TrailingStopLoss) isSellMade() bool {
	return tsl.Order.Status == "done"
}
