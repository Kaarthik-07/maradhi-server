package handler

import (
	"net/http"

	"github.com/yourusername/maradhi-api/internal/model"
)

func (h *Handler) ListTasks(w http.ResponseWriter, r *http.Request) {
	uid := userID(r)
	filter := model.TaskFilter{
		Status:   model.TaskStatus(r.URL.Query().Get("status")),
		Priority: model.Priority(r.URL.Query().Get("priority")),
		Due:      r.URL.Query().Get("due"),
	}
	tasks, err := h.tasks.List(r.Context(), uid, filter)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to fetch tasks")
		return
	}
	writeJSON(w, http.StatusOK, model.APIResponse[[]model.Task]{Data: tasks, Count: len(tasks)})
}

func (h *Handler) CreateTask(w http.ResponseWriter, r *http.Request) {
	uid := userID(r)
	var req model.CreateTaskRequest
	if !decode(w, r, &req) {
		return
	}
	if req.Title == "" {
		writeError(w, http.StatusBadRequest, "title is required")
		return
	}
	if req.Priority == "" {
		req.Priority = model.PriorityMedium
	}
	task, err := h.tasks.Create(r.Context(), uid, req)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create task")
		return
	}
	writeJSON(w, http.StatusCreated, model.APIResponse[model.Task]{Data: task})
}

func (h *Handler) GetTask(w http.ResponseWriter, r *http.Request) {
	uid := userID(r)
	task, err := h.tasks.GetByID(r.Context(), uid, r.PathValue("id"))
	if err != nil {
		if err == model.ErrNotFound {
			writeError(w, http.StatusNotFound, "task not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to fetch task")
		return
	}
	writeJSON(w, http.StatusOK, model.APIResponse[model.Task]{Data: task})
}

func (h *Handler) UpdateTask(w http.ResponseWriter, r *http.Request) {
	uid := userID(r)
	var req model.UpdateTaskRequest
	if !decode(w, r, &req) {
		return
	}
	task, err := h.tasks.Update(r.Context(), uid, r.PathValue("id"), req)
	if err != nil {
		if err == model.ErrNotFound {
			writeError(w, http.StatusNotFound, "task not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to update task")
		return
	}
	writeJSON(w, http.StatusOK, model.APIResponse[model.Task]{Data: task})
}

func (h *Handler) DeleteTask(w http.ResponseWriter, r *http.Request) {
	if err := h.tasks.Delete(r.Context(), userID(r), r.PathValue("id")); err != nil {
		if err == model.ErrNotFound {
			writeError(w, http.StatusNotFound, "task not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to delete task")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) ListTags(w http.ResponseWriter, r *http.Request) {
	tags, err := h.tasks.ListTags(r.Context(), userID(r))
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to fetch tags")
		return
	}
	writeJSON(w, http.StatusOK, model.APIResponse[[]model.Tag]{Data: tags, Count: len(tags)})
}

func (h *Handler) CreateTag(w http.ResponseWriter, r *http.Request) {
	uid := userID(r)
	var req model.CreateTagRequest
	if !decode(w, r, &req) {
		return
	}
	if req.Name == "" {
		writeError(w, http.StatusBadRequest, "tag name is required")
		return
	}
	if req.Color == "" {
		req.Color = "#888888"
	}
	tag, err := h.tasks.CreateTag(r.Context(), uid, req)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create tag")
		return
	}
	writeJSON(w, http.StatusCreated, model.APIResponse[model.Tag]{Data: tag})
}

func (h *Handler) DeleteTag(w http.ResponseWriter, r *http.Request) {
	if err := h.tasks.DeleteTag(r.Context(), userID(r), r.PathValue("id")); err != nil {
		if err == model.ErrNotFound {
			writeError(w, http.StatusNotFound, "tag not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to delete tag")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
