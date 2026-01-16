-- Migration: Create skills table
-- Skills and certifications

CREATE TABLE IF NOT EXISTS skills (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    
    name TEXT NOT NULL,
    category TEXT,               -- e.g., 'programming', 'language', 'soft_skill'
    proficiency_level TEXT CHECK (proficiency_level IS NULL OR proficiency_level IN (
        'beginner', 'intermediate', 'advanced', 'expert'
    )),
    years_of_experience INTEGER,
    
    -- For certifications
    is_certification BOOLEAN DEFAULT FALSE,
    issuing_organization TEXT,
    issue_date DATE,
    expiry_date DATE,
    credential_url TEXT,
    
    imported_from TEXT,
    
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    
    UNIQUE(user_id, name)
);

CREATE INDEX IF NOT EXISTS idx_skills_user_id ON skills(user_id);
CREATE INDEX IF NOT EXISTS idx_skills_category ON skills(user_id, category);

COMMENT ON TABLE skills IS 'Skills and certifications for CV generation';
