package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type User struct {
	ID           string
	Username     string
	Email        *string
	PasswordHash string
	IsAdmin      bool
	CreatedAt    time.Time
}

type UserRepository struct{ db *pgxpool.Pool }

func NewUserRepository(db *pgxpool.Pool) *UserRepository { return &UserRepository{db: db} }

func (r *UserRepository) Create(ctx context.Context, username, email, passwordHash string, isAdmin bool) (User, error) {
	var u User
	var emailArg *string
	if email != "" { emailArg = &email }
	err := r.db.QueryRow(ctx,
		`INSERT INTO users(username,email,password_hash,is_admin)
		 VALUES($1,$2,$3,$4)
		 RETURNING id,username,email,password_hash,is_admin,created_at`,
		username, emailArg, passwordHash, isAdmin,
	).Scan(&u.ID, &u.Username, &u.Email, &u.PasswordHash, &u.IsAdmin, &u.CreatedAt)
	if err != nil { return User{}, fmt.Errorf("users.Create: %w", err) }
	return u, nil
}

func (r *UserRepository) GetByUsername(ctx context.Context, username string) (User, error) {
	var u User
	err := r.db.QueryRow(ctx,
		`SELECT id,username,email,password_hash,is_admin,created_at
		 FROM users WHERE username=$1`, username,
	).Scan(&u.ID, &u.Username, &u.Email, &u.PasswordHash, &u.IsAdmin, &u.CreatedAt)
	if err != nil { return User{}, fmt.Errorf("not found") }
	return u, nil
}

func (r *UserRepository) GetByID(ctx context.Context, id string) (User, error) {
	var u User
	err := r.db.QueryRow(ctx,
		`SELECT id,username,email,password_hash,is_admin,created_at
		 FROM users WHERE id=$1`, id,
	).Scan(&u.ID, &u.Username, &u.Email, &u.PasswordHash, &u.IsAdmin, &u.CreatedAt)
	if err != nil { return User{}, fmt.Errorf("not found") }
	return u, nil
}
