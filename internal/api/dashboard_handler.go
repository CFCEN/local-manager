package api

import (
	"net/http"

	"github.com/gorilla/mux"

	"local-dev-workspace/internal/service"
)

type DashboardHandler struct {
	svc *service.DashboardService
}

func NewDashboardHandler(svc *service.DashboardService) *DashboardHandler {
	return &DashboardHandler{svc: svc}
}

func registerDashboardRoutes(r *mux.Router, h *DashboardHandler) {
	r.HandleFunc("/dashboard/summary", h.Summary).Methods("GET")
}

func (h *DashboardHandler) Summary(w http.ResponseWriter, r *http.Request) {
	summary, err := h.svc.Summary()
	if err != nil {
		WriteError(w, http.StatusInternalServerError, ErrCodeInternal, err.Error())
		return
	}
	WriteSuccess(w, http.StatusOK, summary)
}
