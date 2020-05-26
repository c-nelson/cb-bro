package main

import (
	"log"
	"net/http"
	"os"
	"time"

	cbp "github.com/preichenberger/go-coinbasepro/v2"
)

func main() {

	sec := 10
	windowSize := 10
	currency := "BTC"
	tslPercent := 0.05

	// set up a log file in the same directory
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

	// config the TSL order
	tsl := TrailingStopLoss{
		client:     client,
		currency:   currency,
		tslPercent: tslPercent,
		waitTime:   sec,
		windowSize: windowSize,
	}

	tsl.Run()

}
