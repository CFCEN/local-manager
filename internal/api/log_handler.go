package api

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"

	"local-dev-workspace/internal/model"
	"local-dev-workspace/internal/service"
)

type LogHandler struct {
	svc *service.LogService
}

func NewLogHandler(svc *service.LogService) *LogHandler {
	return &LogHandler{svc: svc}
}

func registerLogRoutes(r *mux.Router, h *LogHandler) {
	r.HandleFunc("/log-groups", h.ListGroups).Methods("GET")
	r.HandleFunc("/log-groups", h.CreateGroup).Methods("POST")
	r.HandleFunc("/log-groups/{id}", h.UpdateGroup).Methods("PUT")
	r.HandleFunc("/log-groups/{id}", h.DeleteGroup).Methods("DELETE")

	r.HandleFunc("/log-files", h.ListFiles).Methods("GET")
	r.HandleFunc("/log-files", h.CreateFile).Methods("POST")
	r.HandleFunc("/log-files/{id}", h.GetFile).Methods("GET")
	r.HandleFunc("/log-files/{id}", h.UpdateFile).Methods("PUT")
	r.HandleFunc("/log-files/{id}", h.DeleteFile).Methods("DELETE")
	r.HandleFunc("/log-files/{id}/content", h.GetContent).Methods("GET")
}

// --- Groups ---

func (h *LogHandler) ListGroups(w http.ResponseWriter, r *http.Request) {
	items, err := h.svc.ListGroups()
	if err != nil {
		WriteError(w, http.StatusInternalServerError, ErrCodeInternal, err.Error())
		return
	}
	WriteList(w, items, int64(len(items)))
}

func (h *LogHandler) CreateGroup(w http.ResponseWriter, r *http.Request) {
	var req model.CreateLogGroupRequest
	if err := DecodeBody(r, &req); err != nil {
		WriteError(w, http.StatusBadRequest, ErrCodeBadRequest, "invalid request body")
		return
	}
	if req.Name == "" {
		WriteError(w, http.StatusBadRequest, ErrCodeBadRequest, "name is required")
		return
	}
	g, err := h.svc.CreateGroup(req)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, ErrCodeInternal, err.Error())
		return
	}
	WriteSuccess(w, http.StatusCreated, g)
}

func (h *LogHandler) UpdateGroup(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		WriteError(w, http.StatusBadRequest, ErrCodeBadRequest, "invalid id")
		return
	}
	var req model.UpdateLogGroupRequest
	if err := DecodeBody(r, &req); err != nil {
		WriteError(w, http.StatusBadRequest, ErrCodeBadRequest, "invalid request body")
		return
	}
	g, err := h.svc.UpdateGroup(id, req)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, ErrCodeInternal, err.Error())
		return
	}
	WriteSuccess(w, http.StatusOK, g)
}

func (h *LogHandler) DeleteGroup(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		WriteError(w, http.StatusBadRequest, ErrCodeBadRequest, "invalid id")
		return
	}
	if err := h.svc.DeleteGroup(id); err != nil {
		WriteError(w, http.StatusInternalServerError, ErrCodeInternal, err.Error())
		return
	}
	WriteSuccess(w, http.StatusOK, map[string]string{"status": "deleted"})
}

// --- Files ---

func (h *LogHandler) ListFiles(w http.ResponseWriter, r *http.Request) {
	items, err := h.svc.ListFiles()
	if err != nil {
		WriteError(w, http.StatusInternalServerError, ErrCodeInternal, err.Error())
		return
	}
	WriteList(w, items, int64(len(items)))
}

func (h *LogHandler) GetFile(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		WriteError(w, http.StatusBadRequest, ErrCodeBadRequest, "invalid id")
		return
	}
	f, err := h.svc.GetFile(id)
	if err != nil {
		WriteError(w, http.StatusNotFound, ErrCodeLogFileNotFound, err.Error())
		return
	}
	WriteSuccess(w, http.StatusOK, f)
}

func (h *LogHandler) CreateFile(w http.ResponseWriter, r *http.Request) {
	var req model.CreateLogFileRequest
	if err := DecodeBody(r, &req); err != nil {
		WriteError(w, http.StatusBadRequest, ErrCodeBadRequest, "invalid request body")
		return
	}
	if req.Name == "" || req.Path == "" {
		WriteError(w, http.StatusBadRequest, ErrCodeBadRequest, "name and path are required")
		return
	}
	f, err := h.svc.CreateFile(req)
	if err != nil {
		WriteError(w, http.StatusBadRequest, ErrCodeFileNotFound, err.Error())
		return
	}
	WriteSuccess(w, http.StatusCreated, f)
}

func (h *LogHandler) UpdateFile(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		WriteError(w, http.StatusBadRequest, ErrCodeBadRequest, "invalid id")
		return
	}
	var req model.UpdateLogFileRequest
	if err := DecodeBody(r, &req); err != nil {
		WriteError(w, http.StatusBadRequest, ErrCodeBadRequest, "invalid request body")
		return
	}
	f, err := h.svc.UpdateFile(id, req)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, ErrCodeInternal, err.Error())
		return
	}
	WriteSuccess(w, http.StatusOK, f)
}

func (h *LogHandler) DeleteFile(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		WriteError(w, http.StatusBadRequest, ErrCodeBadRequest, "invalid id")
		return
	}
	if err := h.svc.DeleteFile(id); err != nil {
		WriteError(w, http.StatusInternalServerError, ErrCodeInternal, err.Error())
		return
	}
	WriteSuccess(w, http.StatusOK, map[string]string{"status": "deleted"})
}

func (h *LogHandler) GetContent(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		WriteError(w, http.StatusBadRequest, ErrCodeBadRequest, "invalid id")
		return
	}
	tailStr := r.URL.Query().Get("tail")
	tail := 0
	if tailStr != "" {
		if v, err := strconv.Atoi(tailStr); err == nil {
			tail = v
		}
	}
	// Clamp requested line count to a sane range (0 = whole file).
	if tail < 0 {
		tail = 0
	}
	if tail > 10000 {
		tail = 10000
	}
	content, err := h.svc.GetContent(id, tail)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrLogFileNotFound):
			WriteError(w, http.StatusNotFound, ErrCodeLogFileNotFound, err.Error())
		case errors.Is(err, service.ErrLogFileOnDiskMissing):
			WriteError(w, http.StatusNotFound, ErrCodeFileNotFound, err.Error())
		case errors.Is(err, service.ErrLogFileBinary):
			WriteError(w, http.StatusBadRequest, ErrCodeFileNotReadable, err.Error())
		default:
			WriteError(w, http.StatusInternalServerError, ErrCodeInternal, err.Error())
		}
		return
	}
	WriteSuccess(w, http.StatusOK, content)
}
