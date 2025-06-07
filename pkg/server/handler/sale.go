package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/IlyushaZ/not-back-contest/pkg/model"
	"github.com/IlyushaZ/not-back-contest/pkg/service"
)

func SaleListPage(svc service.Sale) http.HandlerFunc {
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

		var resp ListPageResp[model.Sale]

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
