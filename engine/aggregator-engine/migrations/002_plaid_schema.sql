CREATE TABLE plaid_items (
    item_id VARCHAR(128) PRIMARY KEY,
    user_id UUID NOT NULL,
    access_token TEXT NOT NULL, -- Encrypted at rest using your crypto package
    institution_id VARCHAR(64) NOT NULL,
    institution_name VARCHAR(128),
    created_at TIMESTAMPTZ DEFAULT NOW(),
    
    UNIQUE(user_id, institution_id)
);