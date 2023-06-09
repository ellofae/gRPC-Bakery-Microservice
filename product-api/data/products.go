package data

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"regexp"
	"time"

	protos "github.com/ellofae/gRPC-Bakery-Microservice/currency/protos/currency"
	"github.com/go-playground/validator"
	"github.com/hashicorp/go-hclog"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Product data type structure
// swagger:model
type Product struct {
	// ID for the product
	//
	// required: true
	// min: 1
	ID          int     `json:"id"`
	Title       string  `json:"title" validate:"required,title"`
	Description string  `json:"description" validate:"required,description"`
	Price       float64 `json:"price" validate:"gt=0"`
	SKU         string  `json:"sku" validate:"required,sku"`
	CreatedOn   string  `json:"-"`
	UpdatedOn   string  `json:"-"`
	DeletedOn   string  `json:"-"`
}

// Validation
func (p *Product) Validate() error {
	validate := validator.New()
	validate.RegisterValidation("sku", validateSKU)
	validate.RegisterValidation("title", validateTitle)
	validate.RegisterValidation("description", validateDescription)
	return validate.Struct(p)
}

func validateSKU(fl validator.FieldLevel) bool {
	re := regexp.MustCompile("[a-z]+-[a-z]+-[a-z]+")
	matches := re.FindAllString(fl.Field().String(), -1)

	if len(matches) != 1 {
		return false
	}
	return true
}

func validateDescription(fl validator.FieldLevel) bool {
	re := regexp.MustCompile("[a-zA-Z0-9-]+")
	matches := re.FindAllString(fl.Field().String(), -1)

	if len(matches) == 0 {
		return false
	}
	return true
}

func validateTitle(fl validator.FieldLevel) bool {
	re := regexp.MustCompile("^[^0-9]+$")
	matches := re.FindAllString(fl.Field().String(), -1)

	if len(matches) != 1 {
		return false
	}
	return true
}

//

// ProductsDB type
type ProductsDB struct {
	currency protos.CurrencyClient
	log      hclog.Logger
	rates    map[string]float64
	client   protos.Currency_SubscribeRatesClient
}

func NewProductsDB(c protos.CurrencyClient, l hclog.Logger) *ProductsDB {
	pb := &ProductsDB{c, l, make(map[string]float64), nil}

	go pb.handleUpdates()
	return pb
}

func (p *ProductsDB) handleUpdates() {
	sub, err := p.currency.SubscribeRates(context.Background())
	if err != nil {
		p.log.Error("Unable to subscribe for rates", "error", err)
		return
	}
	p.client = sub

	for {
		rr, err := sub.Recv()

		if grpcError := rr.GetError(); grpcError != nil {
			p.log.Error("Error subscribing for rates", "error", err)
			continue
		}

		if resp := rr.GetRateResponse(); resp != nil {
			p.log.Info("Recieved updated rate from server", "dest", resp.GetDestination().String())
			if err != nil {
				p.log.Error("Error receiving message", "error", err)
				return
			}

			p.rates[resp.Destination.String()] = resp.Rate
		}
	}
}

func (p *ProductsDB) GetProducts(currency string) (Products, error) {
	if currency == "" {
		return productList, nil
	}

	rate, err := p.getRate(currency)
	if err != nil {
		p.log.Error("Unable to get rate", "error", err)
		return nil, err
	}

	prods := Products{}
	for _, p := range productList {
		np := *p
		np.Price = np.Price * rate
		prods = append(prods, &np)
	}

	return prods, nil
}

func (p *ProductsDB) GetProductByID(id int, currency string) (*Product, error) {
	i := findIndexByProductID(id)
	if id == -1 {
		return nil, fmt.Errorf("there is no product with id %d", id)
	}

	if currency == "" {
		return nil, fmt.Errorf("incorrect currency for the currency rate request: %s", currency)
	}

	rate, err := p.getRate(currency)
	if err != nil {
		p.log.Error("Unable to get rate", "error", err)
		return nil, err
	}

	np := *productList[i]
	np.Price = np.Price * rate

	return &np, nil
}

func (p *ProductsDB) AddProduct(prod *Product) {
	prod.ID = getProductID()
	productList = append(productList, prod)
}

func (p *ProductsDB) UpdateData(id int, prod *Product) error {
	pos, err := getProductPosition(id)
	if err != nil {
		return err
	}

	prod.ID = id
	productList[pos] = prod

	return nil
}

// Products type
type Products []*Product

func (p *Product) ToJSON(w io.Writer) error {
	encoder := json.NewEncoder(w)
	return encoder.Encode(p)
}

func (p *Products) ToJSON(w io.Writer) error {
	encoder := json.NewEncoder(w)
	return encoder.Encode(p)
}

func (p *Product) FromJSON(r io.Reader) error {
	decoder := json.NewDecoder(r)
	return decoder.Decode(p)
}

var ErrProductNotFound = fmt.Errorf("Product was not found")

func getProductPosition(id int) (int, error) {
	for i, p := range productList {
		if p.ID == id {
			return i, nil
		}
	}
	return -1, ErrProductNotFound
}

func getProductID() int {
	p := productList[len(productList)-1]
	return p.ID + 1
}

func findIndexByProductID(id int) int {
	for i, p := range productList {
		if p.ID == id {
			return i
		}
	}

	return -1
}

func (p *ProductsDB) getRate(dest string) (float64, error) {
	// if cached return
	if r, ok := p.rates[dest]; ok {
		return r, nil
	}

	rr := &protos.RateRequest{
		Base:        protos.Currencies(protos.Currencies_value["EUR"]),
		Destination: protos.Currencies(protos.Currencies_value[dest]),
	}

	// get initial rate
	resp, err := p.currency.GetRate(context.Background(), rr)
	if err != nil {
		if s, ok := status.FromError(err); ok {
			md := s.Details()[0].(protos.RateRequest)

			if s.Code() == codes.InvalidArgument {
				return -1, fmt.Errorf("unable to get rate from the currency server, destincation and base currencies cannot be the same, base: %s, dest: %s", md.Base.String(), md.Destination.String())
			}
			return -1, fmt.Errorf("unable to get rate from the currency server, base: %s, dest: %s", md.Base.String(), md.Destination.String())
		}

		return -1, err
	}

	p.rates[dest] = resp.Rate // update cache

	// subscribe for updated
	p.client.Send(rr)

	return resp.Rate, err
}

var productList = []*Product{
	&Product{
		ID:          1,
		Title:       "Chocolate cake",
		Description: "A fluffy cake made of Alpine dark chocolate",
		Price:       5.99,
		SKU:         "soag214f",
		CreatedOn:   time.Now().UTC().String(),
		UpdatedOn:   time.Now().UTC().String(),
		DeletedOn:   time.Now().UTC().String(),
	},
	&Product{
		ID:          2,
		Title:       "Brownie",
		Description: "A tasty chocolate make with berries",
		Price:       1.99,
		SKU:         "fas412a",
		CreatedOn:   time.Now().UTC().String(),
		UpdatedOn:   time.Now().UTC().String(),
		DeletedOn:   time.Now().UTC().String(),
	},
	&Product{
		ID:          3,
		Title:       "Croissant",
		Description: "A crunchy delisious make of bread and vanilla",
		Price:       2.99,
		SKU:         "opf123h",
		CreatedOn:   time.Now().UTC().String(),
		UpdatedOn:   time.Now().UTC().String(),
		DeletedOn:   time.Now().UTC().String(),
	},
}
