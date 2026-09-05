package repository

import (
	"aggregator-engine/internal/domain"
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

// GetConnectionSummaries aggregates the user's broker accounts by institution for the Manage Connections UI.
func (r *PostgresUserRepo) GetConnectionSummaries(ctx context.Context, userID string) ([]domain.ConnectionSummary, error) {
	query := `
		SELECT 
			broker, 
			institution_name, 
			COUNT(account_id) as total_accounts, 
			is_active
		FROM broker_accounts
		WHERE user_id = $1
		GROUP BY broker, institution_name, is_active
		ORDER BY broker DESC, institution_name ASC
	`

	rows, err := r.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var summaries []domain.ConnectionSummary
	for rows.Next() {
		var s domain.ConnectionSummary

		err := rows.Scan(
			&s.Broker,
			&s.InstitutionName,
			&s.TotalAccounts,
			&s.IsActive,
		)
		if err != nil {
			return nil, err
		}

		summaries = append(summaries, s)
	}

	// Guarantee an empty JSON array instead of null
	if summaries == nil {
		summaries = []domain.ConnectionSummary{}
	}

	return summaries, nil
}
