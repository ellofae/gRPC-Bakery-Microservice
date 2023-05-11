package main

import (
	"net"
	"os"

	"github.com/ellofae/gRPC-Bakery-Microservice/currency/data"
	protos "github.com/ellofae/gRPC-Bakery-Microservice/currency/protos/currency"
	"github.com/ellofae/gRPC-Bakery-Microservice/currency/server"
	hclog "github.com/hashicorp/go-hclog"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

func main() {
	log := hclog.Default()

	rates, err := data.NewRates(log)
	if err != nil {
		log.Error("Unable to generate rates", "error", err)
		os.Exit(1)
	}

	gs := grpc.NewServer()
	cs := server.NewCurrency(rates, log)

	protos.RegisterCurrencyServer(gs, cs)

	// Reflection API support setting
	reflection.Register(gs)

	// Specifing a port:
	l, err := net.Listen("tcp", ":9092")
	if err != nil {
		log.Error("Unable to listen", "error", err)
		os.Exit(1)
	}

	gs.Serve(l)
}
