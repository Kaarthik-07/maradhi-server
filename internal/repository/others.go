package repository

import (
	"context"
	"fmt"
	"strconv"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/yourusername/maradhi-api/internal/model"
)

type BucketRepository struct{ db *pgxpool.Pool }
func NewBucketRepository(db *pgxpool.Pool) *BucketRepository { return &BucketRepository{db: db} }

func (r *BucketRepository) List(ctx context.Context, userID, category string) ([]model.BucketListItem, error) {
	q := `SELECT id,user_id,title,description,category,is_done,completed_at,created_at
	      FROM bucket_list_items WHERE user_id=$1`
	args := []any{userID}
	if category != "" { q += " AND category=$2"; args = append(args, category) }
	q += " ORDER BY is_done ASC, created_at DESC"

	rows, err := r.db.Query(ctx, q, args...)
	if err != nil { return nil, err }
	defer rows.Close()
	var items []model.BucketListItem
	for rows.Next() {
		var b model.BucketListItem
		rows.Scan(&b.ID, &b.UserID, &b.Title, &b.Description, &b.Category, &b.IsDone, &b.CompletedAt, &b.CreatedAt)
		items = append(items, b)
	}
	if items == nil { items = []model.BucketListItem{} }
	return items, rows.Err()
}

func (r *BucketRepository) Create(ctx context.Context, userID string, req model.CreateBucketItemRequest) (model.BucketListItem, error) {
	var b model.BucketListItem
	err := r.db.QueryRow(ctx,
		`INSERT INTO bucket_list_items(user_id,title,description,category) VALUES($1,$2,$3,$4)
		 RETURNING id,user_id,title,description,category,is_done,completed_at,created_at`,
		userID, req.Title, req.Description, req.Category,
	).Scan(&b.ID, &b.UserID, &b.Title, &b.Description, &b.Category, &b.IsDone, &b.CompletedAt, &b.CreatedAt)
	return b, err
}

func (r *BucketRepository) Update(ctx context.Context, userID, id string, req model.UpdateBucketItemRequest) (model.BucketListItem, error) {
	completedExpr := ""
	if req.IsDone != nil && *req.IsDone { completedExpr = ",completed_at=NOW()" }
	if req.IsDone != nil && !*req.IsDone { completedExpr = ",completed_at=NULL" }

	var b model.BucketListItem
	err := r.db.QueryRow(ctx, fmt.Sprintf(`
		UPDATE bucket_list_items SET
			title=COALESCE($3,title), description=COALESCE($4,description),
			category=COALESCE($5,category), is_done=COALESCE($6,is_done) %s
		WHERE id=$1 AND user_id=$2
		RETURNING id,user_id,title,description,category,is_done,completed_at,created_at`,
		completedExpr), id, userID, req.Title, req.Description, req.Category, req.IsDone,
	).Scan(&b.ID, &b.UserID, &b.Title, &b.Description, &b.Category, &b.IsDone, &b.CompletedAt, &b.CreatedAt)
	return b, err
}

func (r *BucketRepository) Delete(ctx context.Context, userID, id string) error {
	tag, err := r.db.Exec(ctx, "DELETE FROM bucket_list_items WHERE id=$1 AND user_id=$2", id, userID)
	if err != nil { return err }
	if tag.RowsAffected() == 0 { return model.ErrNotFound }
	return nil
}

type NoteRepository struct{ db *pgxpool.Pool }
func NewNoteRepository(db *pgxpool.Pool) *NoteRepository { return &NoteRepository{db: db} }

func (r *NoteRepository) List(ctx context.Context, userID, tag, search string) ([]model.Note, error) {
	q := `SELECT id,user_id,title,content,tag_label,created_at,updated_at FROM notes WHERE user_id=$1`
	args := []any{userID}; n := 2
	if tag != "" { q += fmt.Sprintf(" AND tag_label=$%d", n); args = append(args, tag); n++ }
	if search != "" { q += fmt.Sprintf(" AND (title ILIKE $%d OR content ILIKE $%d)", n, n); args = append(args, "%"+search+"%"); n++ }
	q += " ORDER BY updated_at DESC"

	rows, err := r.db.Query(ctx, q, args...)
	if err != nil { return nil, err }
	defer rows.Close()
	var notes []model.Note
	for rows.Next() {
		var n model.Note
		rows.Scan(&n.ID, &n.UserID, &n.Title, &n.Content, &n.TagLabel, &n.CreatedAt, &n.UpdatedAt)
		notes = append(notes, n)
	}
	if notes == nil { notes = []model.Note{} }
	return notes, rows.Err()
}

func (r *NoteRepository) GetByID(ctx context.Context, userID, id string) (model.Note, error) {
	var n model.Note
	err := r.db.QueryRow(ctx,
		`SELECT id,user_id,title,content,tag_label,created_at,updated_at FROM notes WHERE id=$1 AND user_id=$2`,
		id, userID).Scan(&n.ID, &n.UserID, &n.Title, &n.Content, &n.TagLabel, &n.CreatedAt, &n.UpdatedAt)
	if err != nil { return model.Note{}, model.ErrNotFound }
	return n, nil
}

func (r *NoteRepository) Create(ctx context.Context, userID string, req model.CreateNoteRequest) (model.Note, error) {
	var n model.Note
	err := r.db.QueryRow(ctx,
		`INSERT INTO notes(user_id,title,content,tag_label) VALUES($1,$2,$3,$4)
		 RETURNING id,user_id,title,content,tag_label,created_at,updated_at`,
		userID, req.Title, req.Content, req.TagLabel,
	).Scan(&n.ID, &n.UserID, &n.Title, &n.Content, &n.TagLabel, &n.CreatedAt, &n.UpdatedAt)
	return n, err
}

func (r *NoteRepository) Update(ctx context.Context, userID, id string, req model.UpdateNoteRequest) (model.Note, error) {
	var n model.Note
	err := r.db.QueryRow(ctx,
		`UPDATE notes SET title=COALESCE($3,title), content=COALESCE($4,content),
		 tag_label=COALESCE($5,tag_label), updated_at=NOW()
		 WHERE id=$1 AND user_id=$2
		 RETURNING id,user_id,title,content,tag_label,created_at,updated_at`,
		id, userID, req.Title, req.Content, req.TagLabel,
	).Scan(&n.ID, &n.UserID, &n.Title, &n.Content, &n.TagLabel, &n.CreatedAt, &n.UpdatedAt)
	return n, err
}

func (r *NoteRepository) Delete(ctx context.Context, userID, id string) error {
	tag, err := r.db.Exec(ctx, "DELETE FROM notes WHERE id=$1 AND user_id=$2", id, userID)
	if err != nil { return err }
	if tag.RowsAffected() == 0 { return model.ErrNotFound }
	return nil
}

type HabitRepository struct{ db *pgxpool.Pool }
func NewHabitRepository(db *pgxpool.Pool) *HabitRepository { return &HabitRepository{db: db} }

func (r *HabitRepository) ListWithWeekLogs(ctx context.Context, userID string) ([]model.Habit, error) {
	rows, err := r.db.Query(ctx, `
		SELECT h.id, h.user_id, h.name, h.created_at,
		       COALESCE(ARRAY_AGG(hl.logged_date::text ORDER BY hl.logged_date)
		                FILTER (WHERE hl.logged_date IS NOT NULL), '{}')
		FROM habits h
		LEFT JOIN habit_logs hl ON hl.habit_id=h.id AND hl.user_id=h.user_id
			AND hl.logged_date >= date_trunc('week', CURRENT_DATE)
			AND hl.logged_date < date_trunc('week', CURRENT_DATE) + INTERVAL '7 days'
		WHERE h.user_id=$1
		GROUP BY h.id ORDER BY h.created_at ASC`, userID)
	if err != nil { return nil, err }
	defer rows.Close()
	var habits []model.Habit
	for rows.Next() {
		var h model.Habit
		rows.Scan(&h.ID, &h.UserID, &h.Name, &h.CreatedAt, &h.WeekLogs)
		habits = append(habits, h)
	}
	if habits == nil { habits = []model.Habit{} }
	return habits, rows.Err()
}

func (r *HabitRepository) Create(ctx context.Context, userID string, req model.CreateHabitRequest) (model.Habit, error) {
	var h model.Habit
	err := r.db.QueryRow(ctx,
		"INSERT INTO habits(user_id,name) VALUES($1,$2) RETURNING id,user_id,name,created_at",
		userID, req.Name).Scan(&h.ID, &h.UserID, &h.Name, &h.CreatedAt)
	h.WeekLogs = []string{}
	return h, err
}

func (r *HabitRepository) Delete(ctx context.Context, userID, id string) error {
	tag, err := r.db.Exec(ctx, "DELETE FROM habits WHERE id=$1 AND user_id=$2", id, userID)
	if err != nil { return err }
	if tag.RowsAffected() == 0 { return model.ErrNotFound }
	return nil
}

func (r *HabitRepository) Log(ctx context.Context, userID, habitID string, req model.LogHabitRequest) error {
	if req.Done {
		_, err := r.db.Exec(ctx,
			`INSERT INTO habit_logs(habit_id,user_id,logged_date) VALUES($1,$2,$3)
			 ON CONFLICT (habit_id,user_id,logged_date) DO NOTHING`,
			habitID, userID, req.Date)
		return err
	}
	_, err := r.db.Exec(ctx,
		"DELETE FROM habit_logs WHERE habit_id=$1 AND user_id=$2 AND logged_date=$3",
		habitID, userID, req.Date)
	return err
}

type FocusRepository struct{ db *pgxpool.Pool }
func NewFocusRepository(db *pgxpool.Pool) *FocusRepository { return &FocusRepository{db: db} }

func (r *FocusRepository) List(ctx context.Context, userID, date string) ([]model.FocusSession, error) {
	q := `SELECT id,user_id,duration_minutes,task_note,started_at,ended_at,created_at
	      FROM focus_sessions WHERE user_id=$1`
	args := []any{userID}
	if date != "" { q += " AND started_at::date=$2"; args = append(args, date) }
	q += " ORDER BY started_at DESC LIMIT 50"
	rows, err := r.db.Query(ctx, q, args...)
	if err != nil { return nil, err }
	defer rows.Close()
	var sessions []model.FocusSession
	for rows.Next() {
		var s model.FocusSession
		rows.Scan(&s.ID, &s.UserID, &s.DurationMinutes, &s.TaskNote, &s.StartedAt, &s.EndedAt, &s.CreatedAt)
		sessions = append(sessions, s)
	}
	if sessions == nil { sessions = []model.FocusSession{} }
	return sessions, rows.Err()
}

func (r *FocusRepository) Create(ctx context.Context, userID string, req model.CreateFocusSessionRequest) (model.FocusSession, error) {
	var s model.FocusSession
	err := r.db.QueryRow(ctx,
		`INSERT INTO focus_sessions(user_id,duration_minutes,task_note,started_at,ended_at)
		 VALUES($1,$2,$3,$4,$5) RETURNING id,user_id,duration_minutes,task_note,started_at,ended_at,created_at`,
		userID, req.DurationMinutes, req.TaskNote, req.StartedAt, req.EndedAt,
	).Scan(&s.ID, &s.UserID, &s.DurationMinutes, &s.TaskNote, &s.StartedAt, &s.EndedAt, &s.CreatedAt)
	return s, err
}

type MoodRepository struct{ db *pgxpool.Pool }
func NewMoodRepository(db *pgxpool.Pool) *MoodRepository { return &MoodRepository{db: db} }

func (r *MoodRepository) List(ctx context.Context, userID, limitStr string) ([]model.MoodLog, error) {
	limit := 30
	if n, err := strconv.Atoi(limitStr); err == nil && n > 0 { limit = n }
	rows, err := r.db.Query(ctx,
		`SELECT id,user_id,mood,note,logged_at,created_at FROM mood_logs
		 WHERE user_id=$1 ORDER BY logged_at DESC LIMIT $2`, userID, limit)
	if err != nil { return nil, err }
	defer rows.Close()
	var logs []model.MoodLog
	for rows.Next() {
		var m model.MoodLog
		rows.Scan(&m.ID, &m.UserID, &m.Mood, &m.Note, &m.LoggedAt, &m.CreatedAt)
		logs = append(logs, m)
	}
	if logs == nil { logs = []model.MoodLog{} }
	return logs, rows.Err()
}

func (r *MoodRepository) Create(ctx context.Context, userID string, req model.CreateMoodLogRequest) (model.MoodLog, error) {
	var m model.MoodLog
	err := r.db.QueryRow(ctx,
		`INSERT INTO mood_logs(user_id,mood,note,logged_at) VALUES($1,$2,$3,$4)
		 RETURNING id,user_id,mood,note,logged_at,created_at`,
		userID, string(req.Mood), req.Note, req.LoggedAt,
	).Scan(&m.ID, &m.UserID, &m.Mood, &m.Note, &m.LoggedAt, &m.CreatedAt)
	return m, err
}
