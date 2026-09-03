package repository

import (
	"context"
	"database/sql"

	"aggregator-engine/internal/pkg/crypto"
)

type PlaidItem struct {
	ItemID          string
	UserID          string
	AccessToken     string // Decrypted in memory
	InstitutionID   string
	InstitutionName string // <-- NEW: Added to store the real bank name
}

type PlaidRepository interface {
	SaveItem(ctx context.Context, item PlaidItem) error
	GetItemsForUser(ctx context.Context, userID string) ([]PlaidItem, error)
}

type PostgresPlaidRepo struct {
	db        *sql.DB
	encryptor *crypto.Encryptor
}

func NewPostgresPlaidRepo(db *sql.DB, encryptor *crypto.Encryptor) *PostgresPlaidRepo {
	return &PostgresPlaidRepo{db: db, encryptor: encryptor}
}

func (r *PostgresPlaidRepo) SaveItem(ctx context.Context, item PlaidItem) error {
	encToken, err := r.encryptor.Encrypt(item.AccessToken)
	if err != nil {
		return err
	}

	// UPDATED: Added institution_name to the INSERT.
	// Changed to DO UPDATE so if a user goes through Plaid "Update Mode"
	// to fix a broken connection, the new token and name are safely saved.
	query := `
        INSERT INTO plaid_items (item_id, user_id, access_token, institution_id, institution_name, created_at)
        VALUES ($1, $2, $3, $4, $5, NOW())
        ON CONFLICT (item_id) DO UPDATE SET 
            access_token = EXCLUDED.access_token,
            institution_name = EXCLUDED.institution_name;
    `
	_, err = r.db.ExecContext(ctx, query, item.ItemID, item.UserID, encToken, item.InstitutionID, item.InstitutionName)
	return err
}

func (r *PostgresPlaidRepo) GetItemsForUser(ctx context.Context, userID string) ([]PlaidItem, error) {
	// UPDATED: Added institution_name to the SELECT
	query := `SELECT item_id, access_token, institution_id, institution_name FROM plaid_items WHERE user_id = $1`
	rows, err := r.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []PlaidItem
	for rows.Next() {
		var item PlaidItem
		var encToken string
		var instName sql.NullString // Used to safely handle older database rows where this might be NULL

		// UPDATED: Scan into the new instName variable
		if err := rows.Scan(&item.ItemID, &encToken, &item.InstitutionID, &instName); err != nil {
			return nil, err
		}

		decToken, err := r.encryptor.Decrypt(encToken)
		if err != nil {
			return nil, err // In production, log this and continue to the next item
		}

		item.AccessToken = decToken
		item.UserID = userID

		if instName.Valid {
			item.InstitutionName = instName.String
		} else {
			item.InstitutionName = "Connected Broker" // Fallback for old sandbox rows
		}

		items = append(items, item)
	}
	return items, nil
}
