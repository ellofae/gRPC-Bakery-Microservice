package main

import (
	"net"
	"os"

	protos "github.com/ellofae/gRPC-Bakery-Microservice/currency/protos/currency"
	"github.com/ellofae/gRPC-Bakery-Microservice/currency/server"
	hclog "github.com/hashicorp/go-hclog"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

func main() {
	log := hclog.Default()

	gs := grpc.NewServer()
	cs := server.NewCurrency(log)

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
