package server

import (
	"net/http"
	"time"

	"github.com/IlyushaZ/not-back-contest/pkg/server/handler"
	"github.com/IlyushaZ/not-back-contest/pkg/server/middleware"
	"github.com/IlyushaZ/not-back-contest/pkg/service"
)

const (
	readTimeout  = 5 * time.Second
	writeTimeout = 5 * time.Second
)

func New(addr string, itemSvc service.Item, saleSvc service.Sale) (*http.Server, error) {
	mux := http.NewServeMux()

	mux.Handle("/checkout", handler.ItemCheckout(itemSvc))
	mux.Handle("/purchase", handler.ItemPurchase(itemSvc))
	mux.Handle("/items", handler.ItemListPage(itemSvc))
	mux.Handle("/sales", handler.SaleListPage(saleSvc))

	chain := middleware.Chain{
		middleware.Log,
		middleware.Recovery,
	}

	return &http.Server{
		Addr:    addr,
		Handler: chain.Then(mux),
	}, nil
}
