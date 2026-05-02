package handler

import (
	"encoding/json"
	"net/http"

	"github.com/example/prom-router/internal/models"
	"github.com/example/prom-router/internal/service"
)

type QueryHandler struct {
	queryService service.QueryService
}

func NewQueryHandler(queryService service.QueryService) *QueryHandler {
	return &QueryHandler{
		queryService: queryService,
	}
}

func (h *QueryHandler) HandleQuery(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	req := models.PromQueryRequest{
		Query:   r.FormValue("query"),
		Time:    r.FormValue("time"),
		Timeout: r.FormValue("timeout"),
	}

	if req.Query == "" {
		http.Error(w, `{"status":"error","errorType":"bad_data","error":"missing query"}`, http.StatusBadRequest)
		return
	}

	res, err := h.queryService.Query(r.Context(), req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(res)
}

func (h *QueryHandler) HandleQueryRange(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	req := models.PromQueryRangeRequest{
		Query:   r.FormValue("query"),
		Start:   r.FormValue("start"),
		End:     r.FormValue("end"),
		Step:    r.FormValue("step"),
		Timeout: r.FormValue("timeout"),
	}

	if req.Query == "" || req.Start == "" || req.End == "" || req.Step == "" {
		http.Error(w, `{"status":"error","errorType":"bad_data","error":"missing required parameters"}`, http.StatusBadRequest)
		return
	}

	res, err := h.queryService.QueryRange(r.Context(), req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(res)
}

func (h *QueryHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/api/v1/query", h.HandleQuery)
	mux.HandleFunc("/api/v1/query_range", h.HandleQueryRange)
}
