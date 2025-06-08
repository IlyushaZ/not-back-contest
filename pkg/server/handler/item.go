package handler

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"github.com/IlyushaZ/not-back-contest/pkg/database"
	"github.com/IlyushaZ/not-back-contest/pkg/model"
	"github.com/IlyushaZ/not-back-contest/pkg/service"
)

func ItemCheckout(svc service.Item) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "only POST method allowed", http.StatusMethodNotAllowed)
			return
		}

		q := r.URL.Query()

		userID, err := strconv.Atoi(q.Get("user_id"))
		if err != nil {
			http.Error(w, fmt.Sprintf("can't parse user_id: %v", err), http.StatusBadRequest)
			return
		}

		itemID, err := strconv.Atoi(q.Get("item_id"))
		if err != nil {
			http.Error(w, fmt.Sprintf("can't parse item_id: %v", err), http.StatusBadRequest)
			return
		}

		code, err := svc.Checkout(r.Context(), userID, itemID)
		switch {
		case errors.Is(err, model.ErrSaleExpired) || errors.Is(err, model.ErrItemUnavailable):
			http.Error(w, err.Error(), http.StatusPreconditionFailed)
			return
		case errors.Is(err, service.ErrLimitExceeded):
			http.Error(w, err.Error(), http.StatusTooManyRequests)
			return
		case err != nil:
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		resp := []byte(fmt.Sprintf(`{"code":"%s"}`, code))
		if _, err := w.Write(resp); err != nil {
			http.Error(w, fmt.Sprintf("can't write response: %v", err), http.StatusInternalServerError)
			return
		}
	}
}

func ItemPurchase(svc service.Item) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "only POST method allowed", http.StatusMethodNotAllowed)
			return
		}

		code := r.URL.Query().Get("code")
		if code == "" {
			http.Error(w, "no code provided", http.StatusBadRequest)
			return
		}

		var cc model.CheckoutCode
		if err := cc.FromString(code); err != nil {
			http.Error(w, fmt.Sprintf("invalid code: %v", err), http.StatusBadRequest)
			return
		}

		err := svc.Purchase(r.Context(), cc)
		switch {
		case errors.Is(err, model.ErrCheckoutExpired), errors.Is(err, model.ErrSaleExpired):
			http.Error(w, err.Error(), http.StatusPreconditionFailed)
		case errors.Is(err, database.ErrNotFound):
			http.Error(w, err.Error(), http.StatusNotFound)
		case err != nil:
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}

		return
	}
}

func ItemListPage(svc service.Item) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "only GET method allowed", http.StatusMethodNotAllowed)
			return
		}

		var (
			q        = r.URL.Query()
			pageNum  = service.DefaultPageNum
			pageSize = service.DefaultPageSize
			err      error
		)

		if pn := q.Get("page_num"); pn != "" {
			pageNum, err = strconv.Atoi(q.Get("page_num"))
			if err != nil {
				http.Error(w, fmt.Sprintf("can't parse page_num: %v", err), http.StatusBadRequest)
				return
			}
		}

		if ps := q.Get("page_size"); ps != "" {
			pageSize, err = strconv.Atoi(q.Get("page_size"))
			if err != nil {
				http.Error(w, fmt.Sprintf("can't parse page_size: %v", err), http.StatusBadRequest)
				return
			}
		}

		var resp ListPageResp[model.Item]

		resp.Page, resp.Total, err = svc.ListPage(r.Context(), pageNum, pageSize)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(resp); err != nil {
			http.Error(w, fmt.Sprintf("can't encode response: %v", err), http.StatusInternalServerError)
			return
		}
	}
}
