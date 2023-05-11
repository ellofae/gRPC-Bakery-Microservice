package server

import (
	"context"

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
