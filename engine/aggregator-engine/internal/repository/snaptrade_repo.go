package repository

import (
	"context"
	"database/sql"

	"aggregator-engine/internal/pkg/crypto"
)

type SnapTradeRepo interface {
	GetSecret(ctx context.Context, userID string) (string, error)
	SetSecret(ctx context.Context, userID string, secret string) error
}

type PostgresSnapTradeRepo struct {
	db        *sql.DB
	encryptor *crypto.Encryptor
}

func NewPostgresSnapTradeRepo(db *sql.DB, encryptor *crypto.Encryptor) *PostgresSnapTradeRepo {
	return &PostgresSnapTradeRepo{
		db:        db,
		encryptor: encryptor,
	}
}

func (r *PostgresSnapTradeRepo) GetSecret(ctx context.Context, userID string) (string, error) {
	var encryptedSecret string // Using string to match Base64 DB column
	query := `SELECT user_secret FROM public.snaptrade_users WHERE user_id = $1`

	err := r.db.QueryRowContext(ctx, query, userID).Scan(&encryptedSecret)
	if err != nil {
		if err == sql.ErrNoRows {
			return "", nil // Valid state: User hasn't registered with SnapTrade yet
		}
		return "", err
	}

	// Decrypt the Base64 string back into the raw SnapTrade secret
	return r.encryptor.Decrypt(encryptedSecret)
}

func (r *PostgresSnapTradeRepo) SetSecret(ctx context.Context, userID string, secret string) error {
	// Encrypt returns a Base64 string
	encryptedSecret, err := r.encryptor.Encrypt(secret)
	if err != nil {
		return err
	}

	// Upsert pattern: Insert new, or update if the user already exists
	query := `
		INSERT INTO public.snaptrade_users (user_id, user_secret) 
		VALUES ($1, $2)
		ON CONFLICT (user_id) DO UPDATE SET user_secret = EXCLUDED.user_secret`

	_, err = r.db.ExecContext(ctx, query, userID, encryptedSecret)
	return err
}
