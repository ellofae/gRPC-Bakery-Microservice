package server

import (
	"context"

	protos "github.com/ellofae/gRPC-Bakery-Microservice/currency/protos/currency"
	hclog "github.com/hashicorp/go-hclog"
)

type Currency struct {
	log hclog.Logger
}

func NewCurrency(log hclog.Logger) *Currency {
	return &Currency{log}
}

func (c *Currency) GetRate(ctx context.Context, rr *protos.RateRequest) (*protos.RateResponse, error) {
	c.log.Info("Handle GetRate", "base", rr.GetBase(), "destination", rr.GetDestination())

	return &protos.RateResponse{Rate: 0.5}, nil
}
