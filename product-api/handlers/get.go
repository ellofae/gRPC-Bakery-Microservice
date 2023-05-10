package handlers

import (
	"context"
	"fmt"
	"net/http"
	"strconv"

	protos "github.com/ellofae/gRPC-Bakery-Microservice/currency/protos/currency"

	"github.com/ellofae/RESTful-API-Gorilla/data"
	"github.com/gorilla/mux"
)

// swagger:route GET /products products listProducts
//
// # Lists all products from the data storage
//
// Responses:
// 	200: productsResponse
//  500: productsResponseError

// GetProducts returns the products from the data storage
func (p *Products) GetProducts(rw http.ResponseWriter, r *http.Request) {
	p.l.Println("GET Method")

	lp := data.GetProducts()

	err := lp.ToJSON(rw)
	if err != nil {
		http.Error(rw, "Didn't manage to encode products data", http.StatusInternalServerError)
		return
	}
}

func (p *Products) GetProductByID(rw http.ResponseWriter, r *http.Request) {
	p.l.Println("GET Method")

	vars := mux.Vars(r)
	id := vars["id"]
	if id == "" {
		http.Error(rw, "[Error] Wrong URI", http.StatusBadRequest)
		return
	}

	idInteger, err := strconv.Atoi(id)
	if err != nil {
		http.Error(rw, "[Error] Didn't manage to convert requested ID to integer type", http.StatusInternalServerError)
		return
	}

	productSpec := &data.Product{}
	found := false

	lp := data.GetProducts()
	for _, prod := range lp {
		if prod.ID == idInteger {
			productSpec = prod
			found = true
			break
		}
	}

	if !found {
		p.l.Printf("There is no such product with ID %d\n", idInteger)
		http.Error(rw, fmt.Sprintf("Didn't manage to find the product with ID: %d", idInteger), http.StatusBadRequest)
		return
	}

	// get exchange
	rr := &protos.RateRequest{
		Base:        protos.Currencies(protos.Currencies_value["EUR"]),
		Destination: protos.Currencies(protos.Currencies_value["GBP"]),
	}
	resp, err := p.cc.GetRate(context.Background(), rr)
	if err != nil {
		p.l.Println("[Error] error getting new rate", err)
		http.Error(rw, fmt.Sprintf("Didn't manage to get new rate: %w", err), http.StatusInternalServerError)
		return
	}

	productSpec.Price = productSpec.Price * resp.Rate

	err = productSpec.ToJSON(rw)
	if err != nil {
		http.Error(rw, "[Error] Didn't manage to encode products data", http.StatusInternalServerError)
		return
	}
}
