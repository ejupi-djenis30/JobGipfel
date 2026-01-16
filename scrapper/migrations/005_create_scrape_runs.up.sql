-- Migration: Create scrape_runs table for telemetry

-- Scrape run status enum
DO $$ BEGIN
    CREATE TYPE scrape_status AS ENUM ('running', 'completed', 'failed', 'cancelled');
EXCEPTION
    WHEN duplicate_object THEN NULL;
END $$;

-- Scrape runs table
CREATE TABLE IF NOT EXISTS scrape_runs (
    id BIGSERIAL PRIMARY KEY,
    strategy TEXT NOT NULL DEFAULT 'full',
    start_time TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    end_time TIMESTAMPTZ,
    status scrape_status NOT NULL DEFAULT 'running',
    
    -- Metrics
    jobs_processed INTEGER NOT NULL DEFAULT 0,
    jobs_inserted INTEGER NOT NULL DEFAULT 0,
    jobs_updated INTEGER NOT NULL DEFAULT 0,
    jobs_skipped INTEGER NOT NULL DEFAULT 0,
    pages_scraped INTEGER NOT NULL DEFAULT 0,
    
    -- Filter configuration used
    filters JSONB,
    
    -- Error tracking
    error_log TEXT,
    
    -- Metadata
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Indexes for common queries
CREATE INDEX IF NOT EXISTS idx_scrape_runs_status ON scrape_runs(status);
CREATE INDEX IF NOT EXISTS idx_scrape_runs_start_time ON scrape_runs(start_time DESC);

COMMENT ON TABLE scrape_runs IS 'Telemetry for scrape runs - tracks progress and metrics';
COMMENT ON COLUMN scrape_runs.filters IS 'JSON object of filter parameters used for this run';
