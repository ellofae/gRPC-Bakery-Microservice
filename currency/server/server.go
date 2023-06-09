package server

import (
	"context"
	"io"
	"time"

	"github.com/ellofae/gRPC-Bakery-Microservice/currency/data"
	protos "github.com/ellofae/gRPC-Bakery-Microservice/currency/protos/currency"
	hclog "github.com/hashicorp/go-hclog"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type Currency struct {
	log           hclog.Logger
	rates         *data.ExchangeRates
	subscriptions map[protos.Currency_SubscribeRatesServer][]*protos.RateRequest
}

func NewCurrency(r *data.ExchangeRates, log hclog.Logger) *Currency {
	c := &Currency{log, r, make(map[protos.Currency_SubscribeRatesServer][]*protos.RateRequest)}
	go c.handleUpdates()

	return c
}

func (c *Currency) handleUpdates() {
	ru := c.rates.MonitorRates(5 * time.Second)
	for range ru {
		c.log.Info("Got updated rates")

		// loop over subscribed clients
		for k, v := range c.subscriptions {

			// loop over rates
			for _, rr := range v {
				r, err := c.rates.GetRate(rr.GetBase().String(), rr.GetDestination().String())
				if err != nil {
					c.log.Error("Unable to get updated rate", "base", rr.GetBase().String(), "destination", rr.GetDestination().String())
				}

				err = k.Send(
					&protos.StreamingRateResponse{
						Message: &protos.StreamingRateResponse_RateResponse{
							RateResponse: &protos.RateResponse{Base: rr.Base, Destination: rr.Destination, Rate: r},
						},
					},
				)

				if err != nil {
					c.log.Error("Unable to send updated rates", "base", rr.GetBase().String(), "destination", rr.GetDestination().String())
				}
			}
		}
	}
}

func (c *Currency) GetRate(ctx context.Context, rr *protos.RateRequest) (*protos.RateResponse, error) {
	c.log.Info("Handle GetRate", "base", rr.GetBase(), "destination", rr.GetDestination())

	if rr.Base == rr.Destination {
		err := status.Newf(
			codes.InvalidArgument,
			"Base currency %s cannot be the same as the destination currency %s",
			rr.Base.String(),
			rr.Destination.String(),
		)

		err, wde := err.WithDetails(rr)
		if wde != nil {
			return nil, wde
		}

		return nil, err.Err()
	}

	rate, err := c.rates.GetRate(rr.GetBase().String(), rr.GetDestination().String())
	if err != nil {
		return nil, err
	}

	return &protos.RateResponse{Base: rr.Base, Destination: rr.Destination, Rate: rate}, nil
}

func (c *Currency) SubscribeRates(src protos.Currency_SubscribeRatesServer) error {
	// handle client messages
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
			return err
		}

		c.log.Info("Handle client request", "request", rr)

		rrs, ok := c.subscriptions[src]
		if !ok {
			rrs = []*protos.RateRequest{}
		}

		// check that subscription does not exist
		var validationError *status.Status
		for _, v := range rrs {
			if v.Base == rr.Base && v.Destination == rr.Destination {
				// subscription exists
				validationError := status.Newf(
					codes.AlreadyExists,
					"Unable to subscribe for currency as subscription already exists",
				)

				// add the original request as metadata
				validationError, err = validationError.WithDetails(rr)
				if err != nil {
					c.log.Error("Unable to add metadata to error", "error", err)
					break
				}

				break
			}
		}

		// if a validation error return error and continue
		if validationError != nil {
			src.Send(&protos.StreamingRateResponse{
				Message: &protos.StreamingRateResponse_Error{
					Error: validationError.Proto(),
				},
			},
			)
			continue
		}

		// all ok
		rrs = append(rrs, rr)
		c.subscriptions[src] = rrs
	}

	return nil
}
