package handler

import (
	"net/http"

	"github.com/yourusername/maradhi-api/internal/model"
)

func (h *Handler) ListBucketItems(w http.ResponseWriter, r *http.Request) {
	items, err := h.bucket.List(r.Context(), userID(r), r.URL.Query().Get("category"))
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to fetch bucket list")
		return
	}
	writeJSON(w, http.StatusOK, model.APIResponse[[]model.BucketListItem]{Data: items, Count: len(items)})
}

func (h *Handler) CreateBucketItem(w http.ResponseWriter, r *http.Request) {
	uid := userID(r)
	var req model.CreateBucketItemRequest
	if !decode(w, r, &req) {
		return
	}
	if req.Title == "" {
		writeError(w, http.StatusBadRequest, "title is required")
		return
	}
	if req.Category == "" {
		req.Category = "general"
	}
	item, err := h.bucket.Create(r.Context(), uid, req)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create item")
		return
	}
	writeJSON(w, http.StatusCreated, model.APIResponse[model.BucketListItem]{Data: item})
}

func (h *Handler) UpdateBucketItem(w http.ResponseWriter, r *http.Request) {
	uid := userID(r)
	var req model.UpdateBucketItemRequest
	if !decode(w, r, &req) {
		return
	}
	item, err := h.bucket.Update(r.Context(), uid, r.PathValue("id"), req)
	if err != nil {
		if err == model.ErrNotFound {
			writeError(w, http.StatusNotFound, "item not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to update item")
		return
	}
	writeJSON(w, http.StatusOK, model.APIResponse[model.BucketListItem]{Data: item})
}

func (h *Handler) DeleteBucketItem(w http.ResponseWriter, r *http.Request) {
	if err := h.bucket.Delete(r.Context(), userID(r), r.PathValue("id")); err != nil {
		if err == model.ErrNotFound {
			writeError(w, http.StatusNotFound, "item not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to delete item")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) ListNotes(w http.ResponseWriter, r *http.Request) {
	notes, err := h.notes.List(r.Context(), userID(r), r.URL.Query().Get("tag"), r.URL.Query().Get("search"))
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to fetch notes")
		return
	}
	writeJSON(w, http.StatusOK, model.APIResponse[[]model.Note]{Data: notes, Count: len(notes)})
}

func (h *Handler) CreateNote(w http.ResponseWriter, r *http.Request) {
	uid := userID(r)
	var req model.CreateNoteRequest
	if !decode(w, r, &req) {
		return
	}
	if req.Title == "" {
		writeError(w, http.StatusBadRequest, "title is required")
		return
	}
	note, err := h.notes.Create(r.Context(), uid, req)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create note")
		return
	}
	writeJSON(w, http.StatusCreated, model.APIResponse[model.Note]{Data: note})
}

func (h *Handler) GetNote(w http.ResponseWriter, r *http.Request) {
	note, err := h.notes.GetByID(r.Context(), userID(r), r.PathValue("id"))
	if err != nil {
		if err == model.ErrNotFound {
			writeError(w, http.StatusNotFound, "note not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to fetch note")
		return
	}
	writeJSON(w, http.StatusOK, model.APIResponse[model.Note]{Data: note})
}

func (h *Handler) UpdateNote(w http.ResponseWriter, r *http.Request) {
	uid := userID(r)
	var req model.UpdateNoteRequest
	if !decode(w, r, &req) {
		return
	}
	note, err := h.notes.Update(r.Context(), uid, r.PathValue("id"), req)
	if err != nil {
		if err == model.ErrNotFound {
			writeError(w, http.StatusNotFound, "note not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to update note")
		return
	}
	writeJSON(w, http.StatusOK, model.APIResponse[model.Note]{Data: note})
}

func (h *Handler) DeleteNote(w http.ResponseWriter, r *http.Request) {
	if err := h.notes.Delete(r.Context(), userID(r), r.PathValue("id")); err != nil {
		if err == model.ErrNotFound {
			writeError(w, http.StatusNotFound, "note not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to delete note")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) ListHabits(w http.ResponseWriter, r *http.Request) {
	habits, err := h.habits.ListWithWeekLogs(r.Context(), userID(r))
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to fetch habits")
		return
	}
	writeJSON(w, http.StatusOK, model.APIResponse[[]model.Habit]{Data: habits, Count: len(habits)})
}

func (h *Handler) CreateHabit(w http.ResponseWriter, r *http.Request) {
	uid := userID(r)
	var req model.CreateHabitRequest
	if !decode(w, r, &req) {
		return
	}
	if req.Name == "" {
		writeError(w, http.StatusBadRequest, "habit name is required")
		return
	}
	habit, err := h.habits.Create(r.Context(), uid, req)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create habit")
		return
	}
	writeJSON(w, http.StatusCreated, model.APIResponse[model.Habit]{Data: habit})
}

func (h *Handler) DeleteHabit(w http.ResponseWriter, r *http.Request) {
	if err := h.habits.Delete(r.Context(), userID(r), r.PathValue("id")); err != nil {
		if err == model.ErrNotFound {
			writeError(w, http.StatusNotFound, "habit not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to delete habit")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) LogHabit(w http.ResponseWriter, r *http.Request) {
	uid := userID(r)
	var req model.LogHabitRequest
	if !decode(w, r, &req) {
		return
	}
	if req.Date == "" {
		writeError(w, http.StatusBadRequest, "date is required (YYYY-MM-DD)")
		return
	}
	if err := h.habits.Log(r.Context(), uid, r.PathValue("id"), req); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to log habit")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) ListFocusSessions(w http.ResponseWriter, r *http.Request) {
	sessions, err := h.focus.List(r.Context(), userID(r), r.URL.Query().Get("date"))
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to fetch sessions")
		return
	}
	writeJSON(w, http.StatusOK, model.APIResponse[[]model.FocusSession]{Data: sessions, Count: len(sessions)})
}

func (h *Handler) CreateFocusSession(w http.ResponseWriter, r *http.Request) {
	uid := userID(r)
	var req model.CreateFocusSessionRequest
	if !decode(w, r, &req) {
		return
	}
	if req.DurationMinutes <= 0 {
		writeError(w, http.StatusBadRequest, "duration_minutes must be positive")
		return
	}
	session, err := h.focus.Create(r.Context(), uid, req)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to save session")
		return
	}
	writeJSON(w, http.StatusCreated, model.APIResponse[model.FocusSession]{Data: session})
}

func (h *Handler) ListMoodLogs(w http.ResponseWriter, r *http.Request) {
	logs, err := h.mood.List(r.Context(), userID(r), r.URL.Query().Get("limit"))
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to fetch mood logs")
		return
	}
	writeJSON(w, http.StatusOK, model.APIResponse[[]model.MoodLog]{Data: logs, Count: len(logs)})
}

func (h *Handler) CreateMoodLog(w http.ResponseWriter, r *http.Request) {
	uid := userID(r)
	var req model.CreateMoodLogRequest
	if !decode(w, r, &req) {
		return
	}
	if req.Mood == "" {
		writeError(w, http.StatusBadRequest, "mood is required")
		return
	}
	log, err := h.mood.Create(r.Context(), uid, req)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to save mood log")
		return
	}
	writeJSON(w, http.StatusCreated, model.APIResponse[model.MoodLog]{Data: log})
}
