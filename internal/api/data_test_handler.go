package api

import (
	"net/http"

	"github.com/gorilla/mux"

	"local-dev-workspace/internal/model"
	"local-dev-workspace/internal/service"
)

type DataTestHandler struct {
	svc *service.DataTestService
}

func NewDataTestHandler(svc *service.DataTestService) *DataTestHandler {
	return &DataTestHandler{svc: svc}
}

func registerDataTestRoutes(r *mux.Router, h *DataTestHandler) {
	r.HandleFunc("/data-test/ping", h.Ping).Methods("GET")
	r.HandleFunc("/data-test/summary", h.Summary).Methods("GET")
	r.HandleFunc("/data-test/echo", h.Echo).Methods("POST")
}

func (h *DataTestHandler) Ping(w http.ResponseWriter, r *http.Request) {
	result, err := h.svc.Ping()
	if err != nil {
		WriteError(w, http.StatusInternalServerError, ErrCodeInternal, err.Error())
		return
	}
	WriteSuccess(w, http.StatusOK, result)
}

func (h *DataTestHandler) Summary(w http.ResponseWriter, r *http.Request) {
	result, err := h.svc.Summary()
	if err != nil {
		WriteError(w, http.StatusInternalServerError, ErrCodeInternal, err.Error())
		return
	}
	WriteSuccess(w, http.StatusOK, result)
}

func (h *DataTestHandler) Echo(w http.ResponseWriter, r *http.Request) {
	var req model.DataTestEchoRequest
	if err := DecodeBody(r, &req); err != nil {
		WriteError(w, http.StatusBadRequest, ErrCodeBadRequest, "invalid request body")
		return
	}
	WriteSuccess(w, http.StatusOK, h.svc.Echo(req))
}
