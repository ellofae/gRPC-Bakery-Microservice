package handlers

import (
	"fmt"
	"net/http"
	"strconv"

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
	rw.Header().Add("Content-Type", "application/json")
	p.l.Info("GET Method")

	cur := r.URL.Query().Get("currency")

	lp, err := p.productDB.GetProducts(cur)
	if err != nil {
		p.l.Error("Didn't manage to get the list of products", "error", err)
		return
	}

	err = lp.ToJSON(rw)
	if err != nil {
		http.Error(rw, "Didn't manage to encode products data", http.StatusInternalServerError)
		return
	}
}

// swagger:route GET /products/{id} products listSingleProduct
//
// # Lists all products from the data storage
//
// Responses:
// 	200: productsResponse
//  500: productsResponseError

// GetProducts returns the products from the data storage

func (p *Products) GetProductByID(rw http.ResponseWriter, r *http.Request) {
	rw.Header().Add("Content-Type", "application/json")
	p.l.Info("GET Method")

	vars := mux.Vars(r)
	id := vars["id"]
	if id == "" {
		http.Error(rw, "[Error] Wrong URI", http.StatusBadRequest)
		return
	}

	idInteger, err := strconv.Atoi(id)
	cur := r.URL.Query().Get("currency")

	if err != nil {
		http.Error(rw, "[Error] Didn't manage to convert requested ID to integer type", http.StatusInternalServerError)
		return
	}

	productSpec, err := p.productDB.GetProductByID(idInteger, cur)
	if err != nil {
		http.Error(rw, fmt.Sprintf("Error: %v", err), http.StatusBadRequest)
		return
	}

	err = productSpec.ToJSON(rw)
	if err != nil {
		http.Error(rw, "[Error] Didn't manage to encode products data", http.StatusInternalServerError)
		return
	}
}
