package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/ellofae/RESTful-API-Gorilla/data"
	"github.com/ellofae/RESTful-API-Gorilla/files"
	"github.com/ellofae/RESTful-API-Gorilla/handlers"
	protos "github.com/ellofae/gRPC-Bakery-Microservice/currency/protos/currency"
	"github.com/go-openapi/runtime/middleware"
	gohandlers "github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	hclog "github.com/hashicorp/go-hclog"
	"google.golang.org/grpc"
)

func main() {
	l := hclog.Default()

	// Connection setting
	conn, err := grpc.Dial("localhost:9092", grpc.WithInsecure())
	if err != nil {
		panic(err)
	}
	defer conn.Close()

	// Client creation
	cc := protos.NewCurrencyClient(conn)

	// ProductsDB
	db := data.NewProductsDB(cc, l)

	// Handlers
	ph := handlers.NewProducts(l, db)

	sm := mux.NewRouter()
	getRouter := sm.Methods(http.MethodGet).Subrouter()
	getRouter.HandleFunc("/", ph.GetProducts)
	getRouter.HandleFunc("/{id:[0-9]+}", ph.GetProductByID)

	postRouter := sm.Methods(http.MethodPost).Subrouter()
	postRouter.HandleFunc("/", ph.AddProducts)
	postRouter.Use(ph.MiddlewareValidationForDatatransfer)

	putRouter := sm.Methods(http.MethodPut).Subrouter()
	putRouter.HandleFunc("/{id:[0-9]+}", ph.UpdateData)
	putRouter.Use(ph.MiddlewareValidationForDatatransfer)

	opts := middleware.RedocOpts{SpecURL: "/swagger.yaml"}
	sh := middleware.Redoc(opts, nil)

	getRouter.Handle("/docs", sh)
	getRouter.Handle("/swagger.yaml", http.FileServer(http.Dir("./")))

	// CORS
	ch := gohandlers.CORS(gohandlers.AllowedOrigins([]string{"*"})) // as an open-api

	// Fileserver part setting
	local, err := files.NewLocal("./filestore", 1024)
	if err != nil {
		return
	}

	hf := handlers.NewFilesHandler(l, local)
	fileRouterPost := sm.Methods(http.MethodPost).Subrouter()
	fileRouterPost.HandleFunc("/files/{id:[0-9]+}/{filename:[a-zA-Z]+\\.[a-z]{3}}", hf.ServeHTTP)

	fileRouterGet := sm.Methods(http.MethodGet).Subrouter()
	fileRouterGet.Handle("/files/{id:[0-9]+}/{filename:[a-zA-Z]+\\.[a-z]{3}}", http.StripPrefix("/files/", http.FileServer(http.Dir("./filestore"))))

	srv := &http.Server{
		Addr:         ":9090",
		Handler:      ch(sm),
		IdleTimeout:  120 * time.Second,
		ReadTimeout:  1 * time.Second,
		WriteTimeout: 1 * time.Second,
	}

	go func() {
		l.Info("Starting server on port 9090")
		err := srv.ListenAndServe()
		if err != nil {
			l.Error("Didn't manage to run the server on port 9090", "error", err)
			os.Exit(1)
		}
	}()

	sigChan := make(chan os.Signal)
	signal.Notify(sigChan, os.Interrupt)
	signal.Notify(sigChan, os.Kill)

	sig := <-sigChan
	l.Info("Recived terminate, gracefil shutdown", "signal", sig)

	// Graceful shutdown
	tc, _ := context.WithTimeout(context.Background(), 30*time.Second)
	srv.Shutdown(tc)
}
