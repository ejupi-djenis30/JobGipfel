-- Migration: Create jobs table
-- Flexible schema supporting both scraped (jobroom) and platform-registered jobs

-- Source type enum
DO $$ BEGIN
    CREATE TYPE job_source AS ENUM ('jobroom', 'platform');
EXCEPTION
    WHEN duplicate_object THEN NULL;
END $$;

-- Job status enum
DO $$ BEGIN
    CREATE TYPE job_status AS ENUM ('active', 'inactive', 'expired', 'draft');
EXCEPTION
    WHEN duplicate_object THEN NULL;
END $$;

CREATE TABLE IF NOT EXISTS jobs (
    -- Primary identifier (UUID from jobroom or generated for platform)
    id TEXT PRIMARY KEY,
    
    -- Source tracking
    source job_source NOT NULL DEFAULT 'jobroom',
    
    -- Timestamps
    created_time TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_time TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    
    -- Status
    status job_status NOT NULL DEFAULT 'active',
    
    -- Source system info (for scraped jobs)
    source_system TEXT,
    external_ref TEXT,
    stellennummer_egov TEXT,
    fingerprint TEXT,
    reporting_obligation BOOLEAN NOT NULL DEFAULT FALSE,
    
    -- Relations (nullable for flexibility)
    company_id BIGINT REFERENCES companies(id) ON DELETE SET NULL,
    location_id BIGINT REFERENCES locations(id) ON DELETE SET NULL,
    
    -- Raw data storage (JSON from API for scraped jobs)
    raw_data JSONB,
    
    -- Platform-specific fields (for jobs registered via platform)
    external_url TEXT,
    number_of_positions INTEGER DEFAULT 1,
    
    -- Metadata
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Indexes for common queries
CREATE INDEX IF NOT EXISTS idx_jobs_source ON jobs(source);
CREATE INDEX IF NOT EXISTS idx_jobs_status ON jobs(status);
CREATE INDEX IF NOT EXISTS idx_jobs_company ON jobs(company_id);
CREATE INDEX IF NOT EXISTS idx_jobs_location ON jobs(location_id);
CREATE INDEX IF NOT EXISTS idx_jobs_created_time ON jobs(created_time DESC);
CREATE INDEX IF NOT EXISTS idx_jobs_updated_time ON jobs(updated_time DESC);
CREATE INDEX IF NOT EXISTS idx_jobs_stellennummer ON jobs(stellennummer_egov) WHERE stellennummer_egov IS NOT NULL;

-- GIN index for raw_data JSONB queries
CREATE INDEX IF NOT EXISTS idx_jobs_raw_data ON jobs USING GIN(raw_data);

-- Add comments for documentation
COMMENT ON TABLE jobs IS 'Main jobs table - supports both scraped and platform-registered jobs';
COMMENT ON COLUMN jobs.source IS 'Origin of the job: jobroom (scraped) or platform (registered)';
COMMENT ON COLUMN jobs.raw_data IS 'Original JSON data from job-room.ch API';
