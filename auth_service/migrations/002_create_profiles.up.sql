-- Migration: Create profiles table
-- Personal information for CV generation

CREATE TABLE IF NOT EXISTS profiles (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    
    -- Personal Information
    first_name TEXT,
    last_name TEXT,
    headline TEXT,              -- Professional headline
    summary TEXT,               -- Professional summary/bio
    
    -- Contact Info
    phone TEXT,
    website TEXT,
    linkedin_url TEXT,
    github_url TEXT,
    
    -- Location
    city TEXT,
    country TEXT,
    postal_code TEXT,
    
    -- Preferences
    preferred_job_titles TEXT[],   -- Array of preferred titles
    preferred_locations TEXT[],    -- Array of preferred locations
    salary_expectation_min INTEGER,
    salary_expectation_max INTEGER,
    salary_currency TEXT DEFAULT 'CHF',
    work_authorization TEXT,       -- e.g., 'citizen', 'work_permit', 'visa_required'
    willing_to_relocate BOOLEAN DEFAULT FALSE,
    
    -- Source tracking
    imported_from TEXT,            -- 'linkedin', 'cv_parser', 'manual'
    raw_import_data JSONB,         -- Store original import data
    
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    
    UNIQUE(user_id)
);

CREATE INDEX IF NOT EXISTS idx_profiles_user_id ON profiles(user_id);

COMMENT ON TABLE profiles IS 'User career profiles for CV generation and job matching';
