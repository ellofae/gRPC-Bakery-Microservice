package server

import (
	"context"
	"io"
	"time"

	"github.com/ellofae/gRPC-Bakery-Microservice/currency/data"
	protos "github.com/ellofae/gRPC-Bakery-Microservice/currency/protos/currency"
	hclog "github.com/hashicorp/go-hclog"
)

type Currency struct {
	log   hclog.Logger
	rates *data.ExchangeRates
}

func NewCurrency(r *data.ExchangeRates, log hclog.Logger) *Currency {
	return &Currency{log, r}
}

func (c *Currency) GetRate(ctx context.Context, rr *protos.RateRequest) (*protos.RateResponse, error) {
	c.log.Info("Handle GetRate", "base", rr.GetBase(), "destination", rr.GetDestination())

	rate, err := c.rates.GetRate(rr.GetBase().String(), rr.GetDestination().String())
	if err != nil {
		return nil, err
	}

	return &protos.RateResponse{Rate: rate}, nil
}

func (c *Currency) SubscribeRates(src protos.Currency_SubscribeRatesServer) error {

	// handle client messages
	go func() {
		for {
			rr, err := src.Recv()
			// io.EOF signals that the client has closed the connection
			if err == io.EOF {
				c.log.Info("Client has closed connection")
				break
			}

			// transport between client and server is unavailable
			if err != nil {
				c.log.Error("Unable to read from the client", "error", err)
				break
			}

			c.log.Info("Handle client request", "request", rr)
		}
	}()

	// handle server responses
	// we block here to keep the connection open
	for {
		// send message back to the client
		err := src.Send(&protos.RateResponse{Rate: 12.1})
		if err != nil {
			return err
		}

		time.Sleep(5 * time.Second)
	}
}
