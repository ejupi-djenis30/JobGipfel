-- Migration: Create applications table
-- Track job applications sent via email or web

CREATE TABLE IF NOT EXISTS applications (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL,  -- References users in auth_service DB
    
    -- Job Information
    job_id TEXT,                    -- Optional reference to jobs table
    job_title TEXT NOT NULL,
    company_name TEXT NOT NULL,
    job_url TEXT,
    
    -- Application Details
    method TEXT NOT NULL CHECK (method IN ('email', 'web')),
    status TEXT NOT NULL DEFAULT 'pending' CHECK (status IN (
        'pending', 'in_progress', 'sent', 'failed', 'retrying'
    )),
    
    -- Email Application
    recipient_email TEXT,
    cover_letter TEXT,
    
    -- Web Application
    screenshot_url TEXT,            -- Screenshot of confirmation
    
    -- Error Handling
    error_message TEXT,
    retry_count INTEGER DEFAULT 0,
    
    -- Timestamps
    sent_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Indexes
CREATE INDEX IF NOT EXISTS idx_applications_user_id ON applications(user_id);
CREATE INDEX IF NOT EXISTS idx_applications_status ON applications(status);
CREATE INDEX IF NOT EXISTS idx_applications_user_status ON applications(user_id, status);
CREATE INDEX IF NOT EXISTS idx_applications_created ON applications(created_at DESC);

COMMENT ON TABLE applications IS 'Job applications sent via email or web automation';
