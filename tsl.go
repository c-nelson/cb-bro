package main

import (
	"fmt"
	"log"
	"time"

	cbp "github.com/preichenberger/go-coinbasepro/v2"
)

// TrailingStopLoss holds information about a trailing stop loss order
type TrailingStopLoss struct {
	sellPrice  float64
	tslPercent float64
	updated    time.Time
	order      cbp.Order
	client     *cbp.Client
	account    cbp.Account
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
	log.Printf("[order]   placed for: %f\n", tsl.sellPrice)
}
