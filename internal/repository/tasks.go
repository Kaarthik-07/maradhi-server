package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/yourusername/maradhi-api/internal/model"
)

type TaskRepository struct{ db *pgxpool.Pool }

func NewTaskRepository(db *pgxpool.Pool) *TaskRepository { return &TaskRepository{db: db} }

func (r *TaskRepository) List(ctx context.Context, userID string, f model.TaskFilter) ([]model.Task, error) {
	where := []string{"t.user_id = $1"}
	args := []any{userID}
	n := 2

	if f.Status != "" {
		where = append(where, fmt.Sprintf("t.status = $%d", n))
		args = append(args, string(f.Status))
		n++
	}
	if f.Priority != "" {
		where = append(where, fmt.Sprintf("t.priority = $%d", n))
		args = append(args, string(f.Priority))
		n++
	}
	if f.Due == "today" {
		where = append(where, fmt.Sprintf("t.due_date::date = $%d", n))
		args = append(args, time.Now().Format("2006-01-02"))
		n++
	} else if f.Due == "overdue" {
		where = append(where, "t.due_date < NOW() AND t.status != 'done'")
	}

	q := fmt.Sprintf(`
		SELECT t.id, t.user_id, t.title, t.notes, t.priority, t.status,
		       t.due_date, t.created_at, t.updated_at,
		       COALESCE(json_agg(json_build_object('id',tg.id,'name',tg.name,'color',tg.color))
		                FILTER (WHERE tg.id IS NOT NULL),'[]') AS tags_json
		FROM tasks t
		LEFT JOIN task_tags tt ON tt.task_id = t.id
		LEFT JOIN tags tg ON tg.id = tt.tag_id
		WHERE %s
		GROUP BY t.id
		ORDER BY CASE t.priority WHEN 'high' THEN 1 WHEN 'medium' THEN 2 ELSE 3 END,
		         t.due_date ASC NULLS LAST, t.created_at DESC`,
		strings.Join(where, " AND "))

	rows, err := r.db.Query(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("tasks.List: %w", err)
	}
	defer rows.Close()
	return scanTasks(rows)
}

func (r *TaskRepository) Create(ctx context.Context, userID string, req model.CreateTaskRequest) (model.Task, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return model.Task{}, err
	}
	defer tx.Rollback(ctx)

	var task model.Task
	err = tx.QueryRow(ctx, `
		INSERT INTO tasks (user_id,title,notes,priority,status,due_date)
		VALUES ($1,$2,$3,$4,'pending',$5)
		RETURNING id,user_id,title,notes,priority,status,due_date,created_at,updated_at`,
		userID, req.Title, req.Notes, string(req.Priority), req.DueDate,
	).Scan(&task.ID, &task.UserID, &task.Title, &task.Notes,
		&task.Priority, &task.Status, &task.DueDate, &task.CreatedAt, &task.UpdatedAt)
	if err != nil {
		return model.Task{}, fmt.Errorf("tasks.Create insert: %w", err)
	}

	if len(req.TagIDs) > 0 {
		if err := attachTags(ctx, tx, task.ID, req.TagIDs); err != nil {
			return model.Task{}, err
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return model.Task{}, err
	}
	return r.GetByID(ctx, userID, task.ID)
}

func (r *TaskRepository) GetByID(ctx context.Context, userID, id string) (model.Task, error) {
	rows, err := r.db.Query(ctx, `
		SELECT t.id, t.user_id, t.title, t.notes, t.priority, t.status,
		       t.due_date, t.created_at, t.updated_at,
		       COALESCE(json_agg(json_build_object('id',tg.id,'name',tg.name,'color',tg.color))
		                FILTER (WHERE tg.id IS NOT NULL),'[]')
		FROM tasks t
		LEFT JOIN task_tags tt ON tt.task_id = t.id
		LEFT JOIN tags tg ON tg.id = tt.tag_id
		WHERE t.id=$1 AND t.user_id=$2
		GROUP BY t.id`, id, userID)
	if err != nil {
		return model.Task{}, err
	}
	defer rows.Close()
	tasks, err := scanTasks(rows)
	if err != nil {
		return model.Task{}, err
	}
	if len(tasks) == 0 {
		return model.Task{}, model.ErrNotFound
	}
	return tasks[0], nil
}

func (r *TaskRepository) Update(ctx context.Context, userID, id string, req model.UpdateTaskRequest) (model.Task, error) {
	set := []string{"updated_at=NOW()"}
	args := []any{}
	n := 1

	if req.Title != nil {
		set = append(set, fmt.Sprintf("title=$%d", n)); args = append(args, *req.Title); n++
	}
	if req.Notes != nil {
		set = append(set, fmt.Sprintf("notes=$%d", n)); args = append(args, *req.Notes); n++
	}
	if req.Priority != nil {
		set = append(set, fmt.Sprintf("priority=$%d", n)); args = append(args, string(*req.Priority)); n++
	}
	if req.Status != nil {
		set = append(set, fmt.Sprintf("status=$%d", n)); args = append(args, string(*req.Status)); n++
	}
	if req.DueDate != nil {
		set = append(set, fmt.Sprintf("due_date=$%d", n)); args = append(args, req.DueDate); n++
	}

	args = append(args, id, userID)
	_, err := r.db.Exec(ctx, fmt.Sprintf("UPDATE tasks SET %s WHERE id=$%d AND user_id=$%d",
		strings.Join(set, ","), n, n+1), args...)
	if err != nil {
		return model.Task{}, fmt.Errorf("tasks.Update: %w", err)
	}

	if req.TagIDs != nil {
		tx, _ := r.db.Begin(ctx)
		tx.Exec(ctx, "DELETE FROM task_tags WHERE task_id=$1", id)
		if len(req.TagIDs) > 0 {
			attachTags(ctx, tx, id, req.TagIDs)
		}
		tx.Commit(ctx)
	}
	return r.GetByID(ctx, userID, id)
}

func (r *TaskRepository) Delete(ctx context.Context, userID, id string) error {
	tag, err := r.db.Exec(ctx, "DELETE FROM tasks WHERE id=$1 AND user_id=$2", id, userID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return model.ErrNotFound
	}
	return nil
}

func (r *TaskRepository) ListTags(ctx context.Context, userID string) ([]model.Tag, error) {
	rows, err := r.db.Query(ctx, "SELECT id,user_id,name,color FROM tags WHERE user_id=$1 ORDER BY name", userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var tags []model.Tag
	for rows.Next() {
		var t model.Tag
		rows.Scan(&t.ID, &t.UserID, &t.Name, &t.Color)
		tags = append(tags, t)
	}
	if tags == nil { tags = []model.Tag{} }
	return tags, rows.Err()
}

func (r *TaskRepository) CreateTag(ctx context.Context, userID string, req model.CreateTagRequest) (model.Tag, error) {
	var t model.Tag
	err := r.db.QueryRow(ctx,
		"INSERT INTO tags(user_id,name,color) VALUES($1,$2,$3) RETURNING id,user_id,name,color",
		userID, req.Name, req.Color).Scan(&t.ID, &t.UserID, &t.Name, &t.Color)
	return t, err
}

func (r *TaskRepository) DeleteTag(ctx context.Context, userID, id string) error {
	tag, err := r.db.Exec(ctx, "DELETE FROM tags WHERE id=$1 AND user_id=$2", id, userID)
	if err != nil { return err }
	if tag.RowsAffected() == 0 { return model.ErrNotFound }
	return nil
}

func attachTags(ctx context.Context, tx pgx.Tx, taskID string, tagIDs []string) error {
	for _, tid := range tagIDs {
		if _, err := tx.Exec(ctx,
			"INSERT INTO task_tags(task_id,tag_id) VALUES($1,$2) ON CONFLICT DO NOTHING",
			taskID, tid); err != nil {
			return fmt.Errorf("attachTags: %w", err)
		}
	}
	return nil
}

func scanTasks(rows pgx.Rows) ([]model.Task, error) {
	var tasks []model.Task
	for rows.Next() {
		var t model.Task
		var tagsJSON []byte
		if err := rows.Scan(&t.ID, &t.UserID, &t.Title, &t.Notes,
			&t.Priority, &t.Status, &t.DueDate,
			&t.CreatedAt, &t.UpdatedAt, &tagsJSON); err != nil {
			return nil, err
		}
		t.Tags = []model.Tag{}
		json.Unmarshal(tagsJSON, &t.Tags)
		tasks = append(tasks, t)
	}
	if tasks == nil { tasks = []model.Task{} }
	return tasks, rows.Err()
}
