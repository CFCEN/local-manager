package api

import (
	"net/http"

	"github.com/gorilla/mux"

	"local-dev-workspace/internal/service"
)

type WorkspaceHandler struct {
	svc *service.WorkspaceService
}

func NewWorkspaceHandler(svc *service.WorkspaceService) *WorkspaceHandler {
	return &WorkspaceHandler{svc: svc}
}

func registerWorkspaceRoutes(r *mux.Router, h *WorkspaceHandler) {
	r.HandleFunc("/workspace", h.GetInfo).Methods("GET")
}

func (h *WorkspaceHandler) GetInfo(w http.ResponseWriter, r *http.Request) {
	info, err := h.svc.GetInfo()
	if err != nil {
		WriteError(w, http.StatusInternalServerError, ErrCodeInternal, err.Error())
		return
	}
	WriteSuccess(w, http.StatusOK, info)
}
