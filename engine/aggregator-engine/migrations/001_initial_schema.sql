CREATE TABLE broker_accounts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL,
    broker VARCHAR(32) NOT NULL,       -- 'SCHWAB', 'IBKR', 'TRADIER'
    account_id VARCHAR(64) NOT NULL,   -- Broker-native account ID
    is_active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    
    -- A user can only link a specific broker account once
    UNIQUE(user_id, broker, account_id)
);

CREATE TABLE positions (
    id VARCHAR(128) NOT NULL,          -- Broker-native position or contract ID
    user_id UUID NOT NULL,
    broker VARCHAR(32) NOT NULL,
    account_id VARCHAR(64) NOT NULL,
    
    symbol VARCHAR(32) NOT NULL,       -- Normalized OCC or Ticker symbol
    asset_class VARCHAR(16) NOT NULL,  -- 'EQUITY' or 'OPTION'
    quantity NUMERIC(18, 8) NOT NULL,
    cost_basis NUMERIC(18, 4) NOT NULL,
    market_value NUMERIC(18, 4) NOT NULL,
    unrealized_pl NUMERIC(18, 4) NOT NULL,
    
    -- Option-specific fields (Nullable for equities)
    underlying_symbol VARCHAR(32),
    expiration_date DATE,
    option_type VARCHAR(8),            -- 'CALL' or 'PUT'
    strike_price NUMERIC(18, 4),
    
    last_synced_at TIMESTAMPTZ NOT NULL,

    -- Composite primary key isolates data across users and brokers
    PRIMARY KEY (user_id, broker, account_id, id),
    FOREIGN KEY (user_id, broker, account_id) 
        REFERENCES broker_accounts(user_id, broker, account_id) ON DELETE CASCADE
);