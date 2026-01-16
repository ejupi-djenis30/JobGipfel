-- Migration: Create experiences table
-- Work experience entries

CREATE TABLE IF NOT EXISTS experiences (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    
    -- Position Info
    title TEXT NOT NULL,
    company_name TEXT NOT NULL,
    company_linkedin_url TEXT,
    company_logo_url TEXT,
    
    -- Employment Details
    employment_type TEXT CHECK (employment_type IS NULL OR employment_type IN (
        'full_time', 'part_time', 'contract', 'internship', 'freelance', 'temporary'
    )),
    location TEXT,
    location_type TEXT CHECK (location_type IS NULL OR location_type IN ('onsite', 'remote', 'hybrid')),
    
    -- Duration
    start_date DATE NOT NULL,
    end_date DATE,               -- NULL = current position
    is_current BOOLEAN DEFAULT FALSE,
    
    -- Description
    description TEXT,
    achievements TEXT[],         -- Array of key achievements
    skills_used TEXT[],          -- Skills used in this role
    
    -- Source
    imported_from TEXT,
    
    -- Ordering
    display_order INTEGER DEFAULT 0,
    
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_experiences_user_id ON experiences(user_id);
CREATE INDEX IF NOT EXISTS idx_experiences_current ON experiences(user_id, is_current) WHERE is_current = TRUE;
CREATE INDEX IF NOT EXISTS idx_experiences_dates ON experiences(user_id, start_date DESC);

COMMENT ON TABLE experiences IS 'Work experience entries for CV generation';
