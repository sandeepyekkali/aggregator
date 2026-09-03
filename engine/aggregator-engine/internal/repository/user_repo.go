package repository

import (
	"context"
	"database/sql"
)

type UserRepo interface {
	CreateUser(ctx context.Context, id string, email string) error
}

type PostgresUserRepo struct {
	db *sql.DB
}

func NewPostgresUserRepo(db *sql.DB) *PostgresUserRepo {
	return &PostgresUserRepo{db: db}
}

func (r *PostgresUserRepo) CreateUser(ctx context.Context, id string, email string) error {
	// ON CONFLICT DO NOTHING makes this safely idempotent
	query := `
		INSERT INTO users (id, email) 
		VALUES ($1, $2) 
		ON CONFLICT (id) DO NOTHING
	`
	_, err := r.db.ExecContext(ctx, query, id, email)
	return err
}
