package repository

import (
	"aggregator-engine/internal/domain"
	"context"
	"database/sql"

	"github.com/lib/pq"
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

	query := `
		INSERT INTO transactions (id, user_id, broker, account_id, symbol, date, name, quantity, price, amount, type)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		ON CONFLICT (id) DO UPDATE SET
			name = EXCLUDED.name,
			quantity = EXCLUDED.quantity,
			price = EXCLUDED.price,
			amount = EXCLUDED.amount,
			type = EXCLUDED.type
	`

	for _, t := range txs {
		_, err := tx.ExecContext(ctx, query,
			t.ID, t.UserID, t.Broker, t.AccountID, t.Symbol, t.Date, t.Name, t.Quantity, t.Price, t.Amount, t.Type,
		)
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}

// SyncTransactionWindow upserts new trades and deletes ghost/canceled trades within a date range
func (r *PostgresTransactionRepo) SyncTransactionWindow(ctx context.Context, userID, broker, accountID, startDate, endDate string, txs []domain.Transaction) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	var activeIDs []string
	upsertQuery := `
		INSERT INTO transactions (id, user_id, broker, account_id, symbol, date, name, quantity, price, amount, type)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		ON CONFLICT (id) DO UPDATE SET
			name = EXCLUDED.name, quantity = EXCLUDED.quantity, price = EXCLUDED.price, amount = EXCLUDED.amount, type = EXCLUDED.type;
	`
	for _, t := range txs {
		if _, err := tx.ExecContext(ctx, upsertQuery, t.ID, t.UserID, t.Broker, t.AccountID, t.Symbol, t.Date, t.Name, t.Quantity, t.Price, t.Amount, t.Type); err != nil {
			return err
		}
		activeIDs = append(activeIDs, t.ID)
	}

	// Delete trades in this date window that the broker no longer reports (e.g. pending trades that settled with a new ID)
	deleteQuery := `
		DELETE FROM transactions 
		WHERE user_id = $1 AND broker = $2 AND account_id = $3 
		AND date >= $4 AND date <= $5 
		AND id <> ALL($6)
	`
	if _, err := tx.ExecContext(ctx, deleteQuery, userID, broker, accountID, startDate, endDate, pq.Array(activeIDs)); err != nil {
		return err
	}

	return tx.Commit()
}

// GetUserTransactions reads the ledger back for the frontend, ordered newest first
func (r *PostgresTransactionRepo) GetUserTransactions(ctx context.Context, userID string) ([]domain.Transaction, error) {
	query := `
		SELECT 
			t.id, t.symbol, TO_CHAR(t.date, 'YYYY-MM-DD') AS date, 
			t.name, t.quantity, t.price, t.amount, t.type,
			COALESCE(ba.institution_name, 'Connected Broker') AS institution_name
		FROM transactions t
		JOIN broker_accounts ba 
			ON t.user_id = ba.user_id 
			AND t.broker = ba.broker
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
	query := `SELECT TO_CHAR(MAX(date), 'YYYY-MM-DD') FROM transactions WHERE user_id = $1`

	err := r.db.QueryRowContext(ctx, query, userID).Scan(&latestDate)
	if err != nil {
		return "", err
	}

	if !latestDate.Valid {
		return "", nil
	}

	return latestDate.String, nil
}
