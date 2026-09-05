package repository

import (
	"context"
	"database/sql"

	"aggregator-engine/internal/pkg/crypto"
)

// PostgresSnapTradeRepo handles database operations for Premium tier connections
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

// SaveSecret encrypts and stores the SnapTrade user secret
func (r *PostgresSnapTradeRepo) SaveSecret(ctx context.Context, userID string, secret string) error {
	encSecret, err := r.encryptor.Encrypt(secret)
	if err != nil {
		return err
	}

	// Upsert logic allows recovering a user account if the secret needs to be rotated
	query := `
		INSERT INTO snaptrade_users (user_id, user_secret, created_at)
		VALUES ($1, $2, NOW())
		ON CONFLICT (user_id) DO UPDATE SET 
			user_secret = EXCLUDED.user_secret;
	`

	_, err = r.db.ExecContext(ctx, query, userID, encSecret)
	return err
}

// GetSecret retrieves and decrypts the SnapTrade user secret
func (r *PostgresSnapTradeRepo) GetSecret(ctx context.Context, userID string) (string, error) {
	var encSecret string
	query := `SELECT user_secret FROM snaptrade_users WHERE user_id = $1`

	err := r.db.QueryRowContext(ctx, query, userID).Scan(&encSecret)
	if err != nil {
		if err == sql.ErrNoRows {
			// Safely return an empty string so the handler knows this is a new user
			return "", nil
		}
		return "", err
	}

	decSecret, err := r.encryptor.Decrypt(encSecret)
	if err != nil {
		return "", err
	}

	return decSecret, nil
}
