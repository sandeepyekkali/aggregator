package repository

import (
	"aggregator-engine/internal/domain"
	"context"
	"database/sql"
)

type PostgresTransactionRepo struct {
	db *sql.DB
}

func NewPostgresTransactionRepo(db *sql.DB) *PostgresTransactionRepo {
	return &PostgresTransactionRepo{db: db}
}

func (r *PostgresTransactionRepo) UpsertTransactions(ctx context.Context, txs []domain.Transaction) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Notice how we explicitly map only the 10 columns that exist in the DDL.
	// The in-memory InstitutionName is safely ignored here.
	query := `
		INSERT INTO transactions (id, user_id, account_id, symbol, date, name, quantity, price, amount, type)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		ON CONFLICT (id) DO UPDATE SET
			name = EXCLUDED.name,
			quantity = EXCLUDED.quantity,
			price = EXCLUDED.price,
			amount = EXCLUDED.amount,
			type = EXCLUDED.type
	`

	for _, t := range txs {
		_, err := tx.ExecContext(ctx, query,
			t.ID, t.UserID, t.AccountID, t.Symbol, t.Date, t.Name, t.Quantity, t.Price, t.Amount, t.Type,
		)
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}

// GetUserTransactions reads the ledger back for the frontend, ordered newest first
func (r *PostgresTransactionRepo) GetUserTransactions(ctx context.Context, userID string) ([]domain.Transaction, error) {
	// FIXED: Provider-agnostic JOIN directly on broker_accounts
	query := `
		SELECT 
			t.id, t.symbol, TO_CHAR(t.date, 'YYYY-MM-DD') AS date, 
			t.name, t.quantity, t.price, t.amount, t.type,
			COALESCE(ba.institution_name, 'Connected Broker') AS institution_name
		FROM transactions t
		JOIN broker_accounts ba 
			ON t.user_id = ba.user_id 
			AND t.account_id = ba.account_id
		WHERE t.user_id = $1 
		ORDER BY t.date DESC
	`
	rows, err := r.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var txs []domain.Transaction
	for rows.Next() {
		var t domain.Transaction

		err := rows.Scan(
			&t.ID, &t.Symbol, &t.Date, &t.Name, &t.Quantity,
			&t.Price, &t.Amount, &t.Type, &t.InstitutionName,
		)
		if err != nil {
			return nil, err
		}
		txs = append(txs, t)
	}

	if txs == nil {
		txs = []domain.Transaction{}
	}

	return txs, nil
}

// GetLatestTransactionDate finds the most recent trade date for a user
func (r *PostgresTransactionRepo) GetLatestTransactionDate(ctx context.Context, userID string) (string, error) {
	var latestDate sql.NullString
	// Use TO_CHAR to guarantee the YYYY-MM-DD format Plaid requires
	query := `SELECT TO_CHAR(MAX(date), 'YYYY-MM-DD') FROM transactions WHERE user_id = $1`

	err := r.db.QueryRowContext(ctx, query, userID).Scan(&latestDate)
	if err != nil {
		return "", err
	}

	if !latestDate.Valid {
		return "", nil // Table is empty for this user
	}

	return latestDate.String, nil
}
