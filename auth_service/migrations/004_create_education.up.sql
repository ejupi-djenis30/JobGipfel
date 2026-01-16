-- Migration: Create education table
-- Education entries

CREATE TABLE IF NOT EXISTS education (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    
    -- Institution
    institution_name TEXT NOT NULL,
    institution_logo_url TEXT,
    
    -- Degree Info
    degree TEXT,                 -- e.g., 'Bachelor', 'Master', 'PhD'
    field_of_study TEXT,         -- e.g., 'Computer Science'
    grade TEXT,                  -- e.g., '3.8 GPA', 'First Class Honours'
    
    -- Duration
    start_date DATE,
    end_date DATE,
    is_current BOOLEAN DEFAULT FALSE,
    
    -- Additional
    description TEXT,
    activities TEXT[],           -- Extracurricular activities
    
    -- Source
    imported_from TEXT,
    
    -- Ordering
    display_order INTEGER DEFAULT 0,
    
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_education_user_id ON education(user_id);

COMMENT ON TABLE education IS 'Education entries for CV generation';
