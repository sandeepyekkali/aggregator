package repository

import (
	"context"
	"database/sql"
	"time"

	"aggregator-engine/internal/domain"
	"aggregator-engine/internal/pkg/crypto"
)

type CredentialRepository interface {
	SaveCredentials(ctx context.Context, userID string, broker domain.BrokerProvider, accountID string, accessToken, refreshToken string, expiresAt time.Time) error
	GetAccessToken(ctx context.Context, userID string, broker domain.BrokerProvider, accountID string) (string, error)
}

type PostgresCredentialRepo struct {
	db        *sql.DB
	encryptor *crypto.Encryptor
}

func NewPostgresCredentialRepo(db *sql.DB, encryptor *crypto.Encryptor) *PostgresCredentialRepo {
	return &PostgresCredentialRepo{
		db:        db,
		encryptor: encryptor,
	}
}

func (r *PostgresCredentialRepo) SaveCredentials(ctx context.Context, userID string, broker domain.BrokerProvider, accountID string, accessToken, refreshToken string, expiresAt time.Time) error {
	encAccess, err := r.encryptor.Encrypt(accessToken)
	if err != nil {
		return err
	}

	var encRefresh sql.NullString
	if refreshToken != "" {
		encRef, err := r.encryptor.Encrypt(refreshToken)
		if err != nil {
			return err
		}
		encRefresh = sql.NullString{String: encRef, Valid: true}
	}

	query := `
		INSERT INTO broker_credentials (user_id, broker, account_id, encrypted_access_token, encrypted_refresh_token, token_expires_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, NOW())
		ON CONFLICT (user_id, broker, account_id) DO UPDATE SET
			encrypted_access_token = EXCLUDED.encrypted_access_token,
			encrypted_refresh_token = COALESCE(EXCLUDED.encrypted_refresh_token, broker_credentials.encrypted_refresh_token),
			token_expires_at = EXCLUDED.token_expires_at,
			updated_at = NOW();
	`

	_, err = r.db.ExecContext(ctx, query, userID, string(broker), accountID, encAccess, encRefresh, expiresAt)
	return err
}

func (r *PostgresCredentialRepo) GetAccessToken(ctx context.Context, userID string, broker domain.BrokerProvider, accountID string) (string, error) {
	query := `
		SELECT encrypted_access_token 
		FROM broker_credentials 
		WHERE user_id = $1 AND broker = $2 AND account_id = $3
	`
	var encAccess string
	err := r.db.QueryRowContext(ctx, query, userID, string(broker), accountID).Scan(&encAccess)
	if err != nil {
		return "", err
	}

	return r.encryptor.Decrypt(encAccess)
}
