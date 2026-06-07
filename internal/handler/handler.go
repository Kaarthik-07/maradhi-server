package handler

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/yourusername/maradhi-api/internal/config"
	"github.com/yourusername/maradhi-api/internal/middleware"
	"github.com/yourusername/maradhi-api/internal/model"
	"github.com/yourusername/maradhi-api/internal/repository"
)

type Handler struct {
	db     *pgxpool.Pool
	cfg    *config.Config
	users  *repository.UserRepository
	tasks  *repository.TaskRepository
	bucket *repository.BucketRepository
	notes  *repository.NoteRepository
	habits *repository.HabitRepository
	focus  *repository.FocusRepository
	mood   *repository.MoodRepository
}

func New(db *pgxpool.Pool, cfg *config.Config) *Handler {
	return &Handler{
		db:     db,
		cfg:    cfg,
		users:  repository.NewUserRepository(db),
		tasks:  repository.NewTaskRepository(db),
		bucket: repository.NewBucketRepository(db),
		notes:  repository.NewNoteRepository(db),
		habits: repository.NewHabitRepository(db),
		focus:  repository.NewFocusRepository(db),
		mood:   repository.NewMoodRepository(db),
	}
}

func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	authed := middleware.Auth(h.cfg.JWTSecret)
	p := func(fn http.HandlerFunc) http.HandlerFunc {
		return authed(http.HandlerFunc(fn)).ServeHTTP
	}

	// Public
	mux.HandleFunc("GET /health",         h.Health)
	mux.HandleFunc("POST /auth/login",    h.Login)
	mux.HandleFunc("POST /auth/register", h.Register)

	// Auth
	mux.HandleFunc("GET /auth/me", p(h.Me))

	// Tasks
	mux.HandleFunc("GET /api/v1/tasks",          p(h.ListTasks))
	mux.HandleFunc("POST /api/v1/tasks",         p(h.CreateTask))
	mux.HandleFunc("GET /api/v1/tasks/{id}",     p(h.GetTask))
	mux.HandleFunc("PATCH /api/v1/tasks/{id}",   p(h.UpdateTask))
	mux.HandleFunc("DELETE /api/v1/tasks/{id}",  p(h.DeleteTask))

	// Tags
	mux.HandleFunc("GET /api/v1/tags",           p(h.ListTags))
	mux.HandleFunc("POST /api/v1/tags",          p(h.CreateTag))
	mux.HandleFunc("DELETE /api/v1/tags/{id}",   p(h.DeleteTag))

	// Bucket
	mux.HandleFunc("GET /api/v1/bucket",         p(h.ListBucketItems))
	mux.HandleFunc("POST /api/v1/bucket",        p(h.CreateBucketItem))
	mux.HandleFunc("PATCH /api/v1/bucket/{id}",  p(h.UpdateBucketItem))
	mux.HandleFunc("DELETE /api/v1/bucket/{id}", p(h.DeleteBucketItem))

	// Notes
	mux.HandleFunc("GET /api/v1/notes",          p(h.ListNotes))
	mux.HandleFunc("POST /api/v1/notes",         p(h.CreateNote))
	mux.HandleFunc("GET /api/v1/notes/{id}",     p(h.GetNote))
	mux.HandleFunc("PATCH /api/v1/notes/{id}",   p(h.UpdateNote))
	mux.HandleFunc("DELETE /api/v1/notes/{id}",  p(h.DeleteNote))

	// Habits
	mux.HandleFunc("GET /api/v1/habits",           p(h.ListHabits))
	mux.HandleFunc("POST /api/v1/habits",          p(h.CreateHabit))
	mux.HandleFunc("DELETE /api/v1/habits/{id}",   p(h.DeleteHabit))
	mux.HandleFunc("POST /api/v1/habits/{id}/log", p(h.LogHabit))

	// Focus
	mux.HandleFunc("GET /api/v1/focus",  p(h.ListFocusSessions))
	mux.HandleFunc("POST /api/v1/focus", p(h.CreateFocusSession))

	// Mood
	mux.HandleFunc("GET /api/v1/mood",  p(h.ListMoodLogs))
	mux.HandleFunc("POST /api/v1/mood", p(h.CreateMoodLog))

	// AI
	if h.cfg.AnthropicAPIKey != "" {
		mux.HandleFunc("POST /api/v1/ai/suggest",   p(h.AISuggestTasks))
		mux.HandleFunc("POST /api/v1/ai/summarise", p(h.AISummariseDay))
	}
}

func (h *Handler) Health(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		slog.Error("writeJSON", "error", err)
	}
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, model.APIError{Message: msg})
}

func decode(w http.ResponseWriter, r *http.Request, dst any) bool {
	if err := json.NewDecoder(r.Body).Decode(dst); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body: "+err.Error())
		return false
	}
	return true
}

func userID(r *http.Request) string {
	return middleware.UserIDFromCtx(r.Context())
}
