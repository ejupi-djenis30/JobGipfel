-- Migration: Create child tables for jobs
-- One-to-one: employments, publications, apply_channels
-- One-to-many: job_descriptions, occupations

-- Employments table (1:1 with jobs)
CREATE TABLE IF NOT EXISTS employments (
    job_id TEXT PRIMARY KEY REFERENCES jobs(id) ON DELETE CASCADE,
    start_date DATE,
    end_date DATE,
    short_employment BOOLEAN NOT NULL DEFAULT FALSE,
    immediately BOOLEAN NOT NULL DEFAULT FALSE,
    permanent BOOLEAN NOT NULL DEFAULT FALSE,
    workload_min INTEGER CHECK (workload_min >= 0 AND workload_min <= 100),
    workload_max INTEGER CHECK (workload_max >= 0 AND workload_max <= 100),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_employments_permanent ON employments(permanent);
CREATE INDEX IF NOT EXISTS idx_employments_workload ON employments(workload_min, workload_max);

COMMENT ON TABLE employments IS 'Employment terms and conditions (1:1 with jobs)';

-- Publications table (1:1 with jobs)
CREATE TABLE IF NOT EXISTS publications (
    job_id TEXT PRIMARY KEY REFERENCES jobs(id) ON DELETE CASCADE,
    start_date DATE,
    end_date DATE,
    eures_display BOOLEAN NOT NULL DEFAULT FALSE,
    public_display BOOLEAN NOT NULL DEFAULT TRUE,
    restricted_display BOOLEAN,
    company_anonymous BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_publications_dates ON publications(start_date, end_date);
CREATE INDEX IF NOT EXISTS idx_publications_public ON publications(public_display);

COMMENT ON TABLE publications IS 'Job publication settings (1:1 with jobs)';

-- Apply channels table (1:1 with jobs)
CREATE TABLE IF NOT EXISTS apply_channels (
    job_id TEXT PRIMARY KEY REFERENCES jobs(id) ON DELETE CASCADE,
    raw_post_address TEXT,
    post_address TEXT,
    email_address TEXT,
    phone_number TEXT,
    form_url TEXT,
    additional_info TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

COMMENT ON TABLE apply_channels IS 'Application submission methods (1:1 with jobs)';

-- Job descriptions table (1:many with jobs)
CREATE TABLE IF NOT EXISTS job_descriptions (
    id BIGSERIAL PRIMARY KEY,
    job_id TEXT NOT NULL REFERENCES jobs(id) ON DELETE CASCADE,
    language_iso_code TEXT NOT NULL,
    title TEXT NOT NULL,
    description TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_job_descriptions_job_id ON job_descriptions(job_id);
CREATE INDEX IF NOT EXISTS idx_job_descriptions_language ON job_descriptions(language_iso_code);
CREATE UNIQUE INDEX IF NOT EXISTS idx_job_descriptions_unique ON job_descriptions(job_id, language_iso_code);

COMMENT ON TABLE job_descriptions IS 'Language-specific job titles and descriptions (1:many with jobs)';

-- Occupations table (1:many with jobs)
CREATE TABLE IF NOT EXISTS occupations (
    id BIGSERIAL PRIMARY KEY,
    job_id TEXT NOT NULL REFERENCES jobs(id) ON DELETE CASCADE,
    avam_occupation_code TEXT,
    work_experience TEXT,
    education_code TEXT,
    qualification_code TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_occupations_job_id ON occupations(job_id);
CREATE INDEX IF NOT EXISTS idx_occupations_avam_code ON occupations(avam_occupation_code);

COMMENT ON TABLE occupations IS 'Occupation classifications and requirements (1:many with jobs)';
