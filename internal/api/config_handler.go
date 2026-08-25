package api

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/gorilla/mux"

	"local-dev-workspace/internal/model"
	"local-dev-workspace/internal/service"
)

type ConfigHandler struct {
	svc *service.ConfigService
}

func NewConfigHandler(svc *service.ConfigService) *ConfigHandler {
	return &ConfigHandler{svc: svc}
}

func registerConfigRoutes(r *mux.Router, h *ConfigHandler) {
	r.HandleFunc("/config-files", h.List).Methods("GET")
	r.HandleFunc("/config-files", h.Create).Methods("POST")
	r.HandleFunc("/config-files/{id}", h.Get).Methods("GET")
	r.HandleFunc("/config-files/{id}", h.Update).Methods("PUT")
	r.HandleFunc("/config-files/{id}", h.Delete).Methods("DELETE")
}

func (h *ConfigHandler) List(w http.ResponseWriter, r *http.Request) {
	items, err := h.svc.List()
	if err != nil {
		WriteError(w, http.StatusInternalServerError, ErrCodeInternal, err.Error())
		return
	}
	WriteList(w, items, int64(len(items)))
}

func (h *ConfigHandler) Get(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		WriteError(w, http.StatusBadRequest, ErrCodeBadRequest, "invalid id")
		return
	}
	detail, err := h.svc.GetDetail(id)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			WriteError(w, http.StatusNotFound, ErrCodeFileNotFound, err.Error())
			return
		}
		WriteError(w, http.StatusInternalServerError, ErrCodeInternal, err.Error())
		return
	}
	WriteSuccess(w, http.StatusOK, detail)
}

func (h *ConfigHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req model.CreateConfigRequest
	if err := DecodeBody(r, &req); err != nil {
		WriteError(w, http.StatusBadRequest, ErrCodeBadRequest, "invalid request body")
		return
	}
	if req.Name == "" || req.Path == "" {
		WriteError(w, http.StatusBadRequest, ErrCodeBadRequest, "name and path are required")
		return
	}
	cf, err := h.svc.Create(req)
	if err != nil {
		code := ErrCodeInternal
		msg := err.Error()
		if strings.Contains(msg, "does not exist") || strings.Contains(msg, "not a file") {
			code = ErrCodeFileNotFound
		} else if strings.Contains(msg, "cannot access") {
			code = ErrCodeFileNotReadable
		}
		WriteError(w, http.StatusBadRequest, code, msg)
		return
	}
	WriteSuccess(w, http.StatusCreated, cf)
}

func (h *ConfigHandler) Update(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		WriteError(w, http.StatusBadRequest, ErrCodeBadRequest, "invalid id")
		return
	}

	// Read raw body to decide update type
	body, err := readBody(r)
	if err != nil {
		WriteError(w, http.StatusBadRequest, ErrCodeBadRequest, "cannot read body")
		return
	}

	// Try content update: {"content": "..."}
	var contentReq model.UpdateConfigContentRequest
	if err := json.Unmarshal(body, &contentReq); err == nil && contentReq.Content != "" {
		if err := h.svc.UpdateContent(id, contentReq); err != nil {
			if strings.Contains(err.Error(), "not found") {
				WriteError(w, http.StatusNotFound, ErrCodeFileNotFound, err.Error())
				return
			}
			WriteError(w, http.StatusInternalServerError, ErrCodeInternal, err.Error())
			return
		}
		WriteSuccess(w, http.StatusOK, map[string]string{"status": "updated"})
		return
	}

	// Try metadata update
	var req model.UpdateConfigRequest
	if err := json.Unmarshal(body, &req); err != nil {
		WriteError(w, http.StatusBadRequest, ErrCodeBadRequest, "invalid request body")
		return
	}
	cf, err := h.svc.Update(id, req)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			WriteError(w, http.StatusNotFound, ErrCodeFileNotFound, err.Error())
			return
		}
		WriteError(w, http.StatusInternalServerError, ErrCodeInternal, err.Error())
		return
	}
	WriteSuccess(w, http.StatusOK, cf)
}

func (h *ConfigHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		WriteError(w, http.StatusBadRequest, ErrCodeBadRequest, "invalid id")
		return
	}
	if err := h.svc.Delete(id); err != nil {
		WriteError(w, http.StatusInternalServerError, ErrCodeInternal, err.Error())
		return
	}
	WriteSuccess(w, http.StatusOK, map[string]string{"status": "deleted"})
}
