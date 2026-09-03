package adapter

import (
	"aggregator-engine/internal/domain"
	"context"
	"fmt"
)

// BrokerAdapter defines the standard data ingestion interface all external brokers must implement.
type BrokerAdapter interface {
	FetchAccountBalance(ctx context.Context, accountID string) (*domain.AccountBalance, error)
	FetchPositions(ctx context.Context, accountID string) ([]domain.Position, error)
}

// AdapterFactory handles dynamic routing to the correct broker integration.
type AdapterFactory struct {
	adapters map[domain.BrokerProvider]BrokerAdapter
}

func NewAdapterFactory() *AdapterFactory {
	return &AdapterFactory{
		adapters: make(map[domain.BrokerProvider]BrokerAdapter),
	}
}

func (f *AdapterFactory) Register(provider domain.BrokerProvider, adapter BrokerAdapter) {
	f.adapters[provider] = adapter
}

func (f *AdapterFactory) GetAdapter(provider domain.BrokerProvider) (BrokerAdapter, error) {
	adapter, exists := f.adapters[provider]
	if !exists {
		return nil, fmt.Errorf("unsupported broker provider: %s", provider)
	}
	return adapter, nil
}
