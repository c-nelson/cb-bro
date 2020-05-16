package main

import (
	"fmt"
	cbp "github.com/preichenberger/go-coinbasepro/v2"
	"log"
	"time"
)

// TrailingStopLoss holds information about a trailing stop loss order
type TrailingStopLoss struct {
	sellPrice  float32
	tslPercent float32
	updated    time.Time
	order      cbp.Order
	client     cbp.Client
	accountID  string
	currency   string
}

// ChangeSellPrice calculates the sell price based on the TSL percent
func (tsl *TrailingStopLoss) ChangeSellPrice(market float32) {
	tsl.sellPrice = market * tsl.tslPercent
}

// UpdateTime is used to track the time a change to the TSL was last made
func (tsl *TrailingStopLoss) UpdateTime() {
	tsl.updated = time.Now()
}

// SetCurrency takes the 3 letter symbol for the currency the TSL should trade on
func (tsl *TrailingStopLoss) SetCurrency(c string) {
	tsl.currency = c
}

// CreateOrder creates an order based on the values of the TSL and saves the order to tsl.order
func (tsl *TrailingStopLoss) CreateOrder(sizeOfSell float32) {
	tsl.order = cbp.Order{
		Price:     fmt.Sprintf("%.2f", tsl.sellPrice),
		Size:      fmt.Sprintf("%f", sizeOfSell),
		Side:      "sell",
		ProductID: fmt.Sprintf("%s-USD", tsl.currency),
	}
	savedOrder, err := tsl.client.CreateOrder(&tsl.order)
	if err != nil {
		log.Fatalln(err.Error())
	}
	tsl.order = savedOrder
}

// TODO: cancel orders
