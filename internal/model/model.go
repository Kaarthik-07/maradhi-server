package model

import (
	"errors"
	"time"
)

var ErrNotFound = errors.New("not found")

type Priority   string
type TaskStatus string
type Mood       string

const (
	PriorityHigh   Priority = "high"
	PriorityMedium Priority = "medium"
	PriorityLow    Priority = "low"
)

const (
	TaskStatusPending TaskStatus = "pending"
	TaskStatusDone    TaskStatus = "done"
	TaskStatusDraft   TaskStatus = "draft"
)

const (
	MoodGreat Mood = "great"
	MoodGood  Mood = "good"
	MoodOkay  Mood = "okay"
	MoodLow   Mood = "low"
)

type TaskFilter struct {
	Status   TaskStatus
	Priority Priority
	Due      string
}

type Task struct {
	ID        string     `json:"id"         db:"id"`
	UserID    string     `json:"user_id"    db:"user_id"`
	Title     string     `json:"title"      db:"title"`
	Notes     *string    `json:"notes"      db:"notes"`
	Priority  Priority   `json:"priority"   db:"priority"`
	Status    TaskStatus `json:"status"     db:"status"`
	DueDate   *time.Time `json:"due_date"   db:"due_date"`
	CreatedAt time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt time.Time  `json:"updated_at" db:"updated_at"`
	Tags      []Tag      `json:"tags"       db:"-"`
}

type CreateTaskRequest struct {
	Title    string     `json:"title"`
	Notes    *string    `json:"notes"`
	Priority Priority   `json:"priority"`
	DueDate  *time.Time `json:"due_date"`
	TagIDs   []string   `json:"tag_ids"`
}

type UpdateTaskRequest struct {
	Title    *string     `json:"title"`
	Notes    *string     `json:"notes"`
	Priority *Priority   `json:"priority"`
	Status   *TaskStatus `json:"status"`
	DueDate  *time.Time  `json:"due_date"`
	TagIDs   []string    `json:"tag_ids"`
}

type Tag struct {
	ID     string `json:"id"      db:"id"`
	UserID string `json:"user_id" db:"user_id"`
	Name   string `json:"name"    db:"name"`
	Color  string `json:"color"   db:"color"`
}

type CreateTagRequest struct {
	Name  string `json:"name"`
	Color string `json:"color"`
}

type BucketListItem struct {
	ID          string     `json:"id"           db:"id"`
	UserID      string     `json:"user_id"      db:"user_id"`
	Title       string     `json:"title"        db:"title"`
	Description *string    `json:"description"  db:"description"`
	Category    string     `json:"category"     db:"category"`
	IsDone      bool       `json:"is_done"      db:"is_done"`
	CompletedAt *time.Time `json:"completed_at" db:"completed_at"`
	CreatedAt   time.Time  `json:"created_at"   db:"created_at"`
}

type CreateBucketItemRequest struct {
	Title       string  `json:"title"`
	Description *string `json:"description"`
	Category    string  `json:"category"`
}

type UpdateBucketItemRequest struct {
	Title       *string `json:"title"`
	Description *string `json:"description"`
	Category    *string `json:"category"`
	IsDone      *bool   `json:"is_done"`
}

type Note struct {
	ID        string    `json:"id"         db:"id"`
	UserID    string    `json:"user_id"    db:"user_id"`
	Title     string    `json:"title"      db:"title"`
	Content   string    `json:"content"    db:"content"`
	TagLabel  string    `json:"tag_label"  db:"tag_label"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}

type CreateNoteRequest struct {
	Title    string `json:"title"`
	Content  string `json:"content"`
	TagLabel string `json:"tag_label"`
}

type UpdateNoteRequest struct {
	Title    *string `json:"title"`
	Content  *string `json:"content"`
	TagLabel *string `json:"tag_label"`
}

type Habit struct {
	ID        string    `json:"id"         db:"id"`
	UserID    string    `json:"user_id"    db:"user_id"`
	Name      string    `json:"name"       db:"name"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	WeekLogs  []string  `json:"week_logs"  db:"-"`
}

type HabitLog struct {
	ID         string    `json:"id"          db:"id"`
	HabitID    string    `json:"habit_id"    db:"habit_id"`
	UserID     string    `json:"user_id"     db:"user_id"`
	LoggedDate string    `json:"logged_date" db:"logged_date"`
	CreatedAt  time.Time `json:"created_at"  db:"created_at"`
}

type CreateHabitRequest struct{ Name string `json:"name"` }

type LogHabitRequest struct {
	Date string `json:"date"`
	Done bool   `json:"done"`
}

type FocusSession struct {
	ID              string    `json:"id"               db:"id"`
	UserID          string    `json:"user_id"          db:"user_id"`
	DurationMinutes int       `json:"duration_minutes" db:"duration_minutes"`
	TaskNote        *string   `json:"task_note"        db:"task_note"`
	StartedAt       time.Time `json:"started_at"       db:"started_at"`
	EndedAt         time.Time `json:"ended_at"         db:"ended_at"`
	CreatedAt       time.Time `json:"created_at"       db:"created_at"`
}

type CreateFocusSessionRequest struct {
	DurationMinutes int       `json:"duration_minutes"`
	TaskNote        *string   `json:"task_note"`
	StartedAt       time.Time `json:"started_at"`
	EndedAt         time.Time `json:"ended_at"`
}

type MoodLog struct {
	ID        string    `json:"id"         db:"id"`
	UserID    string    `json:"user_id"    db:"user_id"`
	Mood      Mood      `json:"mood"       db:"mood"`
	Note      *string   `json:"note"       db:"note"`
	LoggedAt  time.Time `json:"logged_at"  db:"logged_at"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
}

type CreateMoodLogRequest struct {
	Mood     Mood      `json:"mood"`
	Note     *string   `json:"note"`
	LoggedAt time.Time `json:"logged_at"`
}

type APIError struct {
	Code    int    `json:"-"`
	Message string `json:"message"`
	Detail  string `json:"detail,omitempty"`
}

func (e *APIError) Error() string { return e.Message }

type APIResponse[T any] struct {
	Data  T   `json:"data"`
	Count int `json:"count,omitempty"`
}
