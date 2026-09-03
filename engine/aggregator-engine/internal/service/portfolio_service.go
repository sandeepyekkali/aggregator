package service

import (
	"context"
	"fmt"
	"time"

	"aggregator-engine/internal/adapter"
	"aggregator-engine/internal/domain"
	"aggregator-engine/internal/repository"
)

type PortfolioService struct {
	factory *adapter.AdapterFactory
	repo    repository.PositionRepository
}

func NewPortfolioService(factory *adapter.AdapterFactory, repo repository.PositionRepository) *PortfolioService {
	return &PortfolioService{
		factory: factory,
		repo:    repo,
	}
}

// SyncAccountState syncs a specific broker account tied to a user.
func (s *PortfolioService) SyncAccountState(ctx context.Context, acc domain.BrokerAccount) error {
	if !acc.IsActive {
		return nil
	}

	// 1. Resolve correct broker adapter dynamically
	brokerAdapter, err := s.factory.GetAdapter(acc.Broker)
	if err != nil {
		return err
	}

	// 2. Fetch raw positions
	positions, err := brokerAdapter.FetchPositions(ctx, acc.AccountID)
	if err != nil {
		return fmt.Errorf("failed fetching positions from %s for account %s: %w", acc.Broker, acc.AccountID, err)
	}

	// 3. Inject strict tenant and broker isolation boundaries before persistence
	now := time.Now().UTC()
	for i := range positions {
		positions[i].UserID = acc.UserID
		positions[i].Broker = acc.Broker
		positions[i].AccountID = acc.AccountID
		positions[i].LastSyncedAt = now
	}

	// 4. Persist normalized data to the database safely
	if err := s.repo.UpsertPositions(ctx, positions); err != nil {
		return fmt.Errorf("failed persisting positions: %w", err)
	}

	return nil
}

// GetAggregatedPortfolio pulls down a unified view of all broker accounts for the user.
func (s *PortfolioService) GetAggregatedPortfolio(ctx context.Context, userID string) ([]domain.Position, error) {
	return s.repo.GetPositionsByUser(ctx, userID)
}
