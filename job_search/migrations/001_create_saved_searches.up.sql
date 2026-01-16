-- Migration: Create saved_searches table
-- Store user saved searches

CREATE TABLE IF NOT EXISTS saved_searches (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL,  -- References users in auth_service
    
    name TEXT NOT NULL,
    query TEXT,
    filters JSONB,
    notify_new BOOLEAN DEFAULT FALSE,
    
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Indexes
CREATE INDEX IF NOT EXISTS idx_saved_searches_user_id ON saved_searches(user_id);

COMMENT ON TABLE saved_searches IS 'User saved job searches';
