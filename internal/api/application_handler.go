package api

import (
	"net/http"
	"strings"

	"github.com/gorilla/mux"

	"local-dev-workspace/internal/model"
	"local-dev-workspace/internal/service"
)

type ApplicationHandler struct {
	svc *service.ApplicationService
}

func NewApplicationHandler(svc *service.ApplicationService) *ApplicationHandler {
	return &ApplicationHandler{svc: svc}
}

func registerApplicationRoutes(r *mux.Router, h *ApplicationHandler) {
	r.HandleFunc("/applications", h.List).Methods("GET")
	r.HandleFunc("/applications", h.Create).Methods("POST")
	r.HandleFunc("/applications/{id}", h.Get).Methods("GET")
	r.HandleFunc("/applications/{id}", h.Update).Methods("PUT")
	r.HandleFunc("/applications/{id}", h.Delete).Methods("DELETE")
	r.HandleFunc("/applications/{id}/start", h.Start).Methods("POST")
	r.HandleFunc("/applications/{id}/stop", h.Stop).Methods("POST")
	r.HandleFunc("/applications/{id}/restart", h.Restart).Methods("POST")
	r.HandleFunc("/applications/{id}/status", h.Status).Methods("GET")
}

func (h *ApplicationHandler) List(w http.ResponseWriter, r *http.Request) {
	items, err := h.svc.List()
	if err != nil {
		WriteError(w, http.StatusInternalServerError, ErrCodeInternal, err.Error())
		return
	}
	WriteList(w, items, int64(len(items)))
}

func (h *ApplicationHandler) Get(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		WriteError(w, http.StatusBadRequest, ErrCodeBadRequest, "invalid id")
		return
	}
	a, err := h.svc.Get(id)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			WriteError(w, http.StatusNotFound, ErrCodeApplicationNotFound, err.Error())
			return
		}
		WriteError(w, http.StatusInternalServerError, ErrCodeInternal, err.Error())
		return
	}
	WriteSuccess(w, http.StatusOK, a)
}

func (h *ApplicationHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req model.CreateApplicationRequest
	if err := DecodeBody(r, &req); err != nil {
		WriteError(w, http.StatusBadRequest, ErrCodeBadRequest, "invalid request body")
		return
	}
	if req.Name == "" || len(req.Commands) == 0 {
		WriteError(w, http.StatusBadRequest, ErrCodeBadRequest, "name and at least one command are required")
		return
	}
	a, err := h.svc.Create(req)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, ErrCodeInternal, err.Error())
		return
	}
	WriteSuccess(w, http.StatusCreated, a)
}

func (h *ApplicationHandler) Update(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		WriteError(w, http.StatusBadRequest, ErrCodeBadRequest, "invalid id")
		return
	}
	var req model.UpdateApplicationRequest
	if err := DecodeBody(r, &req); err != nil {
		WriteError(w, http.StatusBadRequest, ErrCodeBadRequest, "invalid request body")
		return
	}
	a, err := h.svc.Update(id, req)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			WriteError(w, http.StatusNotFound, ErrCodeApplicationNotFound, err.Error())
			return
		}
		WriteError(w, http.StatusInternalServerError, ErrCodeInternal, err.Error())
		return
	}
	WriteSuccess(w, http.StatusOK, a)
}

func (h *ApplicationHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		WriteError(w, http.StatusBadRequest, ErrCodeBadRequest, "invalid id")
		return
	}
	if err := h.svc.Delete(id); err != nil {
		if strings.Contains(err.Error(), "not found") {
			WriteError(w, http.StatusNotFound, ErrCodeApplicationNotFound, err.Error())
			return
		}
		WriteError(w, http.StatusInternalServerError, ErrCodeInternal, err.Error())
		return
	}
	WriteSuccess(w, http.StatusOK, map[string]string{"status": "deleted"})
}

func (h *ApplicationHandler) Start(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		WriteError(w, http.StatusBadRequest, ErrCodeBadRequest, "invalid id")
		return
	}
	commandName := r.URL.Query().Get("command")
	resp, err := h.svc.Start(id, commandName)
	if err != nil {
		code := ErrCodeApplicationStartFailed
		if strings.Contains(err.Error(), "command not found") {
			code = ErrCodeInvalidCommand
		} else if strings.Contains(err.Error(), "not found") {
			code = ErrCodeApplicationNotFound
		} else if strings.Contains(err.Error(), "already running") {
			code = ErrCodeApplicationAlreadyRunning
		}
		WriteError(w, http.StatusBadRequest, code, err.Error())
		return
	}
	WriteSuccess(w, http.StatusOK, resp)
}

func (h *ApplicationHandler) Stop(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		WriteError(w, http.StatusBadRequest, ErrCodeBadRequest, "invalid id")
		return
	}
	if err := h.svc.Stop(id); err != nil {
		code := ErrCodeApplicationStopFailed
		if strings.Contains(err.Error(), "not running") {
			code = ErrCodeApplicationNotRunning
		}
		WriteError(w, http.StatusBadRequest, code, err.Error())
		return
	}
	WriteSuccess(w, http.StatusOK, map[string]string{"status": "stopped"})
}

func (h *ApplicationHandler) Restart(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		WriteError(w, http.StatusBadRequest, ErrCodeBadRequest, "invalid id")
		return
	}
	commandName := r.URL.Query().Get("command")
	resp, err := h.svc.Restart(id, commandName)
	if err != nil {
		WriteError(w, http.StatusBadRequest, ErrCodeApplicationStartFailed, err.Error())
		return
	}
	WriteSuccess(w, http.StatusOK, resp)
}

func (h *ApplicationHandler) Status(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		WriteError(w, http.StatusBadRequest, ErrCodeBadRequest, "invalid id")
		return
	}
	status, err := h.svc.Status(id)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			WriteError(w, http.StatusNotFound, ErrCodeApplicationNotFound, err.Error())
			return
		}
		WriteError(w, http.StatusInternalServerError, ErrCodeInternal, err.Error())
		return
	}
	WriteSuccess(w, http.StatusOK, status)
}
