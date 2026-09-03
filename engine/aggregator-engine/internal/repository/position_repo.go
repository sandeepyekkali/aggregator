package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"aggregator-engine/internal/domain"

	"github.com/lib/pq"
)

type PositionRepository interface {
	UpsertPositions(ctx context.Context, positions []domain.Position) error
	GetPositionsByUser(ctx context.Context, userID string) ([]domain.Position, error)
	SyncAccountPositions(ctx context.Context, userID string, accountID string, positions []domain.Position) error
}

type PostgresPositionRepo struct {
	db *sql.DB
}

func NewPostgresPositionRepo(db *sql.DB) *PostgresPositionRepo {
	return &PostgresPositionRepo{db: db}
}

func (r *PostgresPositionRepo) UpsertPositions(ctx context.Context, positions []domain.Position) error {
	if len(positions) == 0 {
		return nil
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	query := `
		INSERT INTO positions (
			id, user_id, broker, account_id, symbol, asset_class, quantity, cost_basis, 
			market_value, unrealized_pl, underlying_symbol, expiration_date, 
			option_type, strike_price, last_synced_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15
		) ON CONFLICT (user_id, broker, account_id, id) DO UPDATE SET
			quantity = EXCLUDED.quantity,
			cost_basis = EXCLUDED.cost_basis,
			market_value = EXCLUDED.market_value,
			unrealized_pl = EXCLUDED.unrealized_pl,
			last_synced_at = EXCLUDED.last_synced_at;
	`

	stmt, err := tx.PrepareContext(ctx, query)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, p := range positions {
		var underlying, optType *string
		var expDate *time.Time
		var strike *float64

		if p.OptionData != nil {
			underlying = &p.OptionData.UnderlyingSymbol
			exp := p.OptionData.ExpirationDate
			// FIXED: Prevent PostgreSQL 0001-01-01 out-of-range panic
			if !exp.IsZero() {
				expDate = &exp
			}
			ot := string(p.OptionData.OptionType)
			optType = &ot
			strike = &p.OptionData.StrikePrice
		}

		_, err := stmt.ExecContext(ctx,
			p.ID, p.UserID, string(p.Broker), p.AccountID, p.Symbol, string(p.AssetClass),
			p.Quantity, p.CostBasis, p.MarketValue, p.UnrealizedPL,
			underlying, expDate, optType, strike, p.LastSyncedAt,
		)
		if err != nil {
			return fmt.Errorf("failed to upsert position %s: %w", p.Symbol, err)
		}
	}

	return tx.Commit()
}

func (r *PostgresPositionRepo) SyncAccountPositions(ctx context.Context, userID string, accountID string, positions []domain.Position) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	var activeIDs []string

	if len(positions) > 0 {
		query := `
			INSERT INTO positions (
				id, user_id, broker, account_id, symbol, asset_class, quantity, cost_basis, 
				market_value, unrealized_pl, underlying_symbol, expiration_date, 
				option_type, strike_price, last_synced_at
			) VALUES (
				$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15
			) ON CONFLICT (user_id, broker, account_id, id) DO UPDATE SET
				quantity = EXCLUDED.quantity,
				cost_basis = EXCLUDED.cost_basis,
				market_value = EXCLUDED.market_value,
				unrealized_pl = EXCLUDED.unrealized_pl,
				last_synced_at = EXCLUDED.last_synced_at;
		`

		stmt, err := tx.PrepareContext(ctx, query)
		if err != nil {
			return fmt.Errorf("failed to prepare statement: %w", err)
		}
		defer stmt.Close()

		for _, p := range positions {
			var underlying, optType *string
			var expDate *time.Time
			var strike *float64

			if p.OptionData != nil {
				underlying = &p.OptionData.UnderlyingSymbol
				exp := p.OptionData.ExpirationDate
				// FIXED: Prevent PostgreSQL 0001-01-01 out-of-range panic
				if !exp.IsZero() {
					expDate = &exp
				}
				ot := string(p.OptionData.OptionType)
				optType = &ot
				strike = &p.OptionData.StrikePrice
			}

			_, err := stmt.ExecContext(ctx,
				p.ID, p.UserID, string(p.Broker), p.AccountID, p.Symbol, string(p.AssetClass),
				p.Quantity, p.CostBasis, p.MarketValue, p.UnrealizedPL,
				underlying, expDate, optType, strike, p.LastSyncedAt,
			)
			if err != nil {
				return fmt.Errorf("failed to upsert position %s: %w", p.Symbol, err)
			}

			activeIDs = append(activeIDs, p.ID)
		}
	}

	deleteQuery := `
		DELETE FROM positions
		WHERE user_id = $1 
		  AND account_id = $2 
		  AND id <> ALL($3)
	`
	_, err = tx.ExecContext(ctx, deleteQuery, userID, accountID, pq.Array(activeIDs))
	if err != nil {
		return fmt.Errorf("failed to delete ghost positions: %w", err)
	}

	return tx.Commit()
}

func (r *PostgresPositionRepo) GetPositionsByUser(ctx context.Context, userID string) ([]domain.Position, error) {
	// FIXED: Provider-agnostic JOIN on broker_accounts.
	query := `
		SELECT 
			p.id, p.user_id, p.broker, p.account_id, p.symbol, p.asset_class, 
			p.quantity, p.cost_basis, p.market_value, p.unrealized_pl, 
			p.underlying_symbol, p.expiration_date, p.option_type, p.strike_price, p.last_synced_at,
			COALESCE(ba.institution_name, 'Connected Broker') AS institution_name
		FROM positions p
		JOIN broker_accounts ba 
			ON p.user_id = ba.user_id 
			AND p.broker = ba.broker 
			AND p.account_id = ba.account_id
		WHERE p.user_id = $1
	`
	rows, err := r.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var positions []domain.Position
	for rows.Next() {
		var p domain.Position
		var broker string
		var assetClass string
		var underlying, optType sql.NullString
		var expDate sql.NullTime
		var strike sql.NullFloat64

		err := rows.Scan(
			&p.ID, &p.UserID, &broker, &p.AccountID, &p.Symbol, &assetClass,
			&p.Quantity, &p.CostBasis, &p.MarketValue, &p.UnrealizedPL,
			&underlying, &expDate, &optType, &strike, &p.LastSyncedAt,
			&p.InstitutionName,
		)
		if err != nil {
			return nil, err
		}

		p.Broker = domain.BrokerProvider(broker)
		p.AssetClass = domain.AssetClass(assetClass)

		if p.AssetClass == domain.AssetClassOption && underlying.Valid {
			p.OptionData = &domain.OptionDetails{
				UnderlyingSymbol: underlying.String,
				ExpirationDate:   expDate.Time,
				OptionType:       domain.OptionType(optType.String),
				StrikePrice:      strike.Float64,
			}
		}
		positions = append(positions, p)
	}

	if positions == nil {
		positions = []domain.Position{}
	}

	return positions, nil
}
