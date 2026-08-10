package api

import (
	"net/http"
	"strings"

	"github.com/gorilla/mux"

	"local-dev-workspace/internal/model"
	"local-dev-workspace/internal/service"
)

type GitHandler struct {
	svc   *service.GitService
	wsSvc *service.WorkspaceService
}

func NewGitHandler(svc *service.GitService, wsSvc *service.WorkspaceService) *GitHandler {
	return &GitHandler{svc: svc, wsSvc: wsSvc}
}

func registerGitRoutes(r *mux.Router, h *GitHandler) {
	// Projects
	r.HandleFunc("/git/projects", h.ListProjects).Methods("GET")
	r.HandleFunc("/git/projects/scan", h.Scan).Methods("POST")
	r.HandleFunc("/git/projects/batch/group", h.BatchAssignGroup).Methods("POST")
	r.HandleFunc("/git/projects/{id}", h.GetProject).Methods("GET")
	r.HandleFunc("/git/projects/{id}", h.UpdateProject).Methods("PUT")
	r.HandleFunc("/git/projects/{id}", h.DeleteProject).Methods("DELETE")

	// Groups
	r.HandleFunc("/git/groups", h.ListGroups).Methods("GET")
	r.HandleFunc("/git/groups", h.CreateGroup).Methods("POST")
	r.HandleFunc("/git/groups/{id}", h.GetGroup).Methods("GET")
	r.HandleFunc("/git/groups/{id}/projects", h.ListGroupProjects).Methods("GET")
	r.HandleFunc("/git/groups/{id}", h.UpdateGroup).Methods("PUT")
	r.HandleFunc("/git/groups/{id}", h.DeleteGroup).Methods("DELETE")

	// Assign
	r.HandleFunc("/git/projects/{id}/group", h.AssignGroup).Methods("POST")

	// Contexts
	r.HandleFunc("/git/groups/{groupId}/contexts", h.ListContexts).Methods("GET")
	r.HandleFunc("/git/groups/{groupId}/contexts", h.CreateContext).Methods("POST")
	r.HandleFunc("/git/groups/{groupId}/contexts/{contextId}", h.GetContext).Methods("GET")
	r.HandleFunc("/git/groups/{groupId}/contexts/{contextId}", h.UpdateContext).Methods("PUT")
	r.HandleFunc("/git/groups/{groupId}/contexts/{contextId}", h.DeleteContext).Methods("DELETE")

	// Context Files
	r.HandleFunc("/git/groups/{groupId}/contexts/{contextId}/files", h.ListContextFiles).Methods("GET")
	r.HandleFunc("/git/groups/{groupId}/contexts/{contextId}/files/content", h.GetContextFileContent).Methods("GET")
	r.HandleFunc("/git/groups/{groupId}/contexts/{contextId}/files", h.CreateContextFile).Methods("POST")
	r.HandleFunc("/git/groups/{groupId}/contexts/{contextId}/files", h.UpdateContextFile).Methods("PUT")
	r.HandleFunc("/git/groups/{groupId}/contexts/{contextId}/files", h.DeleteContextFile).Methods("DELETE")
}

// --- Projects ---

func (h *GitHandler) ListProjects(w http.ResponseWriter, r *http.Request) {
	items, err := h.svc.ListProjects()
	if err != nil {
		WriteError(w, http.StatusInternalServerError, ErrCodeInternal, err.Error())
		return
	}
	WriteList(w, items, int64(len(items)))
}

func (h *GitHandler) Scan(w http.ResponseWriter, r *http.Request) {
	var req model.ScanRequest
	if err := DecodeBody(r, &req); err != nil {
		WriteError(w, http.StatusBadRequest, ErrCodeBadRequest, "invalid request body")
		return
	}
	if req.Path == "" {
		WriteError(w, http.StatusBadRequest, ErrCodeBadRequest, "path is required")
		return
	}
	results, err := h.svc.Scan(req)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, ErrCodeGitScanFailed, err.Error())
		return
	}
	WriteList(w, results, int64(len(results)))
}

func (h *GitHandler) GetProject(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		WriteError(w, http.StatusBadRequest, ErrCodeBadRequest, "invalid id")
		return
	}
	detail, err := h.svc.GetProjectDetail(id)
	if err != nil {
		WriteError(w, http.StatusNotFound, ErrCodeGitProjectNotFound, err.Error())
		return
	}
	WriteSuccess(w, http.StatusOK, detail)
}

func (h *GitHandler) UpdateProject(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		WriteError(w, http.StatusBadRequest, ErrCodeBadRequest, "invalid id")
		return
	}
	var req model.UpdateGitProjectRequest
	if err := DecodeBody(r, &req); err != nil {
		WriteError(w, http.StatusBadRequest, ErrCodeBadRequest, "invalid request body")
		return
	}
	p, err := h.svc.UpdateProject(id, req)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, ErrCodeInternal, err.Error())
		return
	}
	WriteSuccess(w, http.StatusOK, p)
}

func (h *GitHandler) DeleteProject(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		WriteError(w, http.StatusBadRequest, ErrCodeBadRequest, "invalid id")
		return
	}
	if err := h.svc.DeleteProject(id); err != nil {
		WriteError(w, http.StatusInternalServerError, ErrCodeInternal, err.Error())
		return
	}
	WriteSuccess(w, http.StatusOK, map[string]string{"status": "deleted"})
}

// --- Groups ---

func (h *GitHandler) ListGroups(w http.ResponseWriter, r *http.Request) {
	items, err := h.svc.ListGroups()
	if err != nil {
		WriteError(w, http.StatusInternalServerError, ErrCodeInternal, err.Error())
		return
	}
	WriteList(w, items, int64(len(items)))
}

func (h *GitHandler) GetGroup(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		WriteError(w, http.StatusBadRequest, ErrCodeBadRequest, "invalid id")
		return
	}
	g, err := h.svc.GetGroup(id)
	if err != nil {
		WriteError(w, http.StatusNotFound, ErrCodeGroupNotFound, err.Error())
		return
	}
	WriteSuccess(w, http.StatusOK, g)
}

func (h *GitHandler) CreateGroup(w http.ResponseWriter, r *http.Request) {
	var req model.CreateGitGroupRequest
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

func (h *GitHandler) UpdateGroup(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		WriteError(w, http.StatusBadRequest, ErrCodeBadRequest, "invalid id")
		return
	}
	var req model.UpdateGitGroupRequest
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

func (h *GitHandler) DeleteGroup(w http.ResponseWriter, r *http.Request) {
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

func (h *GitHandler) AssignGroup(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		WriteError(w, http.StatusBadRequest, ErrCodeBadRequest, "invalid id")
		return
	}
	var req model.AssignGroupRequest
	if err := DecodeBody(r, &req); err != nil {
		WriteError(w, http.StatusBadRequest, ErrCodeBadRequest, "invalid request body")
		return
	}
	p, err := h.svc.AssignGroup(id, req)
	if err != nil {
		WriteError(w, http.StatusBadRequest, ErrCodeGroupNotFound, err.Error())
		return
	}
	WriteSuccess(w, http.StatusOK, p)
}

func (h *GitHandler) BatchAssignGroup(w http.ResponseWriter, r *http.Request) {
	var req model.BatchAssignGroupRequest
	if err := DecodeBody(r, &req); err != nil {
		WriteError(w, http.StatusBadRequest, ErrCodeBadRequest, "invalid request body")
		return
	}
	if len(req.ProjectIDs) == 0 {
		WriteError(w, http.StatusBadRequest, ErrCodeBadRequest, "projectIds cannot be empty")
		return
	}
	results, err := h.svc.BatchAssignGroup(req)
	if err != nil {
		if err.Error() == "group not found" {
			WriteError(w, http.StatusNotFound, ErrCodeGroupNotFound, err.Error())
			return
		}
		WriteError(w, http.StatusInternalServerError, ErrCodeInternal, err.Error())
		return
	}
	WriteList(w, results, int64(len(results)))
}

func (h *GitHandler) ListGroupProjects(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		WriteError(w, http.StatusBadRequest, ErrCodeBadRequest, "invalid id")
		return
	}
	items, err := h.svc.ListProjectsByGroup(id)
	if err != nil {
		if err.Error() == "group not found" {
			WriteError(w, http.StatusNotFound, ErrCodeGroupNotFound, err.Error())
			return
		}
		WriteError(w, http.StatusInternalServerError, ErrCodeInternal, err.Error())
		return
	}
	WriteList(w, items, int64(len(items)))
}

// --- Contexts ---

func (h *GitHandler) ListContexts(w http.ResponseWriter, r *http.Request) {
	groupID, err := parseGroupID(r)
	if err != nil {
		WriteError(w, http.StatusBadRequest, ErrCodeBadRequest, "invalid groupId")
		return
	}
	items, err := h.wsSvc.ListContextsByGroup(groupID)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, ErrCodeInternal, err.Error())
		return
	}
	WriteList(w, items, int64(len(items)))
}

func (h *GitHandler) GetContext(w http.ResponseWriter, r *http.Request) {
	contextID, err := parseContextID(r)
	if err != nil {
		WriteError(w, http.StatusBadRequest, ErrCodeBadRequest, "invalid contextId")
		return
	}
	c, err := h.wsSvc.GetContext(contextID)
	if err != nil {
		if err.Error() == "context not found" {
			WriteError(w, http.StatusNotFound, ErrCodeContextNotFound, err.Error())
			return
		}
		WriteError(w, http.StatusInternalServerError, ErrCodeInternal, err.Error())
		return
	}
	WriteSuccess(w, http.StatusOK, c)
}

func (h *GitHandler) CreateContext(w http.ResponseWriter, r *http.Request) {
	groupID, err := parseGroupID(r)
	if err != nil {
		WriteError(w, http.StatusBadRequest, ErrCodeBadRequest, "invalid groupId")
		return
	}
	var req model.CreateContextRequest
	if err := DecodeBody(r, &req); err != nil {
		WriteError(w, http.StatusBadRequest, ErrCodeBadRequest, "invalid request body")
		return
	}
	if req.RAGPath == "" || req.IndexPath == "" {
		WriteError(w, http.StatusBadRequest, ErrCodeBadRequest, "ragPath and indexPath are required")
		return
	}
	c, err := h.wsSvc.CreateContext(groupID, req)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, ErrCodeInternal, err.Error())
		return
	}
	WriteSuccess(w, http.StatusCreated, c)
}

func (h *GitHandler) UpdateContext(w http.ResponseWriter, r *http.Request) {
	contextID, err := parseContextID(r)
	if err != nil {
		WriteError(w, http.StatusBadRequest, ErrCodeBadRequest, "invalid contextId")
		return
	}
	var req model.UpdateContextRequest
	if err := DecodeBody(r, &req); err != nil {
		WriteError(w, http.StatusBadRequest, ErrCodeBadRequest, "invalid request body")
		return
	}
	c, err := h.wsSvc.UpdateContext(contextID, req)
	if err != nil {
		if err.Error() == "context not found" {
			WriteError(w, http.StatusNotFound, ErrCodeContextNotFound, err.Error())
			return
		}
		WriteError(w, http.StatusInternalServerError, ErrCodeInternal, err.Error())
		return
	}
	WriteSuccess(w, http.StatusOK, c)
}

func (h *GitHandler) DeleteContext(w http.ResponseWriter, r *http.Request) {
	contextID, err := parseContextID(r)
	if err != nil {
		WriteError(w, http.StatusBadRequest, ErrCodeBadRequest, "invalid contextId")
		return
	}
	if err := h.wsSvc.DeleteContext(contextID); err != nil {
		if err.Error() == "context not found" {
			WriteError(w, http.StatusNotFound, ErrCodeContextNotFound, err.Error())
			return
		}
		WriteError(w, http.StatusInternalServerError, ErrCodeInternal, err.Error())
		return
	}
	WriteSuccess(w, http.StatusOK, nil)
}

// --- Context Files ---

func (h *GitHandler) ListContextFiles(w http.ResponseWriter, r *http.Request) {
	contextID, err := parseContextID(r)
	if err != nil {
		WriteError(w, http.StatusBadRequest, ErrCodeBadRequest, "invalid contextId")
		return
	}
	fileType := r.URL.Query().Get("type")
	if fileType != "rag" && fileType != "index" {
		WriteError(w, http.StatusBadRequest, ErrCodeBadRequest, "type must be 'rag' or 'index'")
		return
	}
	items, err := h.wsSvc.ListContextFiles(contextID, fileType)
	if err != nil {
		if err.Error() == "context not found" {
			WriteError(w, http.StatusNotFound, ErrCodeContextNotFound, err.Error())
			return
		}
		WriteError(w, http.StatusInternalServerError, ErrCodeInternal, err.Error())
		return
	}
	WriteList(w, items, int64(len(items)))
}

func (h *GitHandler) GetContextFileContent(w http.ResponseWriter, r *http.Request) {
	contextID, err := parseContextID(r)
	if err != nil {
		WriteError(w, http.StatusBadRequest, ErrCodeBadRequest, "invalid contextId")
		return
	}
	fileType := r.URL.Query().Get("type")
	fileName := r.URL.Query().Get("file")
	if fileType != "rag" && fileType != "index" {
		WriteError(w, http.StatusBadRequest, ErrCodeBadRequest, "type must be 'rag' or 'index'")
		return
	}
	if fileName == "" {
		WriteError(w, http.StatusBadRequest, ErrCodeBadRequest, "file parameter is required")
		return
	}
	result, err := h.wsSvc.GetContextFileContent(contextID, fileType, fileName)
	if err != nil {
		msg := err.Error()
		switch {
		case msg == "context not found":
			WriteError(w, http.StatusNotFound, ErrCodeContextNotFound, msg)
		case msg == "file not found":
			WriteError(w, http.StatusNotFound, ErrCodeFileNotFound, msg)
		case strings.HasPrefix(msg, "invalid"):
			WriteError(w, http.StatusBadRequest, ErrCodeInvalidPath, msg)
		case strings.HasPrefix(msg, "file type not supported"):
			WriteError(w, http.StatusBadRequest, ErrCodeFileNotWritable, msg)
		default:
			WriteError(w, http.StatusInternalServerError, ErrCodeInternal, msg)
		}
		return
	}
	WriteSuccess(w, http.StatusOK, result)
}

func (h *GitHandler) CreateContextFile(w http.ResponseWriter, r *http.Request) {
	contextID, err := parseContextID(r)
	if err != nil {
		WriteError(w, http.StatusBadRequest, ErrCodeBadRequest, "invalid contextId")
		return
	}
	var req model.CreateFileRequest
	if err := DecodeBody(r, &req); err != nil {
		WriteError(w, http.StatusBadRequest, ErrCodeBadRequest, "invalid request body")
		return
	}
	if req.Type != "rag" && req.Type != "index" {
		WriteError(w, http.StatusBadRequest, ErrCodeBadRequest, "type must be 'rag' or 'index'")
		return
	}
	if req.FileName == "" || req.Content == "" {
		WriteError(w, http.StatusBadRequest, ErrCodeBadRequest, "fileName and content are required")
		return
	}
	result, err := h.wsSvc.CreateContextFile(contextID, req)
	if err != nil {
		msg := err.Error()
		switch {
		case msg == "context not found":
			WriteError(w, http.StatusNotFound, ErrCodeContextNotFound, msg)
		case msg == "file already exists":
			WriteError(w, http.StatusConflict, ErrCodeFileAlreadyExists, msg)
		case strings.HasPrefix(msg, "invalid") || strings.HasPrefix(msg, "file type not supported") || strings.HasPrefix(msg, "file too large"):
			WriteError(w, http.StatusBadRequest, ErrCodeFileNotWritable, msg)
		case strings.HasPrefix(msg, "failed to write file"):
			WriteError(w, http.StatusInternalServerError, ErrCodeFileNotWritable, msg)
		default:
			WriteError(w, http.StatusInternalServerError, ErrCodeInternal, msg)
		}
		return
	}
	WriteSuccess(w, http.StatusCreated, result)
}

func (h *GitHandler) UpdateContextFile(w http.ResponseWriter, r *http.Request) {
	contextID, err := parseContextID(r)
	if err != nil {
		WriteError(w, http.StatusBadRequest, ErrCodeBadRequest, "invalid contextId")
		return
	}
	var req model.UpdateFileRequest
	if err := DecodeBody(r, &req); err != nil {
		WriteError(w, http.StatusBadRequest, ErrCodeBadRequest, "invalid request body")
		return
	}
	if req.Type != "rag" && req.Type != "index" {
		WriteError(w, http.StatusBadRequest, ErrCodeBadRequest, "type must be 'rag' or 'index'")
		return
	}
	if req.FileName == "" || req.Content == "" {
		WriteError(w, http.StatusBadRequest, ErrCodeBadRequest, "fileName and content are required")
		return
	}
	result, err := h.wsSvc.UpdateContextFile(contextID, req)
	if err != nil {
		msg := err.Error()
		switch {
		case msg == "context not found":
			WriteError(w, http.StatusNotFound, ErrCodeContextNotFound, msg)
		case msg == "file not found":
			WriteError(w, http.StatusNotFound, ErrCodeFileNotFound, msg)
		case strings.HasPrefix(msg, "invalid") || strings.HasPrefix(msg, "file type not supported") || strings.HasPrefix(msg, "file too large"):
			WriteError(w, http.StatusBadRequest, ErrCodeFileNotWritable, msg)
		case strings.HasPrefix(msg, "failed to write file"):
			WriteError(w, http.StatusInternalServerError, ErrCodeFileNotWritable, msg)
		default:
			WriteError(w, http.StatusInternalServerError, ErrCodeInternal, msg)
		}
		return
	}
	WriteSuccess(w, http.StatusOK, result)
}

func (h *GitHandler) DeleteContextFile(w http.ResponseWriter, r *http.Request) {
	contextID, err := parseContextID(r)
	if err != nil {
		WriteError(w, http.StatusBadRequest, ErrCodeBadRequest, "invalid contextId")
		return
	}
	fileType := r.URL.Query().Get("type")
	fileName := r.URL.Query().Get("file")
	if fileType != "rag" && fileType != "index" {
		WriteError(w, http.StatusBadRequest, ErrCodeBadRequest, "type must be 'rag' or 'index'")
		return
	}
	if fileName == "" {
		WriteError(w, http.StatusBadRequest, ErrCodeBadRequest, "file parameter is required")
		return
	}
	if err := h.wsSvc.DeleteContextFile(contextID, fileType, fileName); err != nil {
		msg := err.Error()
		switch {
		case msg == "context not found":
			WriteError(w, http.StatusNotFound, ErrCodeContextNotFound, msg)
		case msg == "file not found":
			WriteError(w, http.StatusNotFound, ErrCodeFileNotFound, msg)
		case strings.HasPrefix(msg, "invalid"):
			WriteError(w, http.StatusBadRequest, ErrCodeInvalidPath, msg)
		default:
			WriteError(w, http.StatusInternalServerError, ErrCodeInternal, msg)
		}
		return
	}
	WriteSuccess(w, http.StatusOK, nil)
}
