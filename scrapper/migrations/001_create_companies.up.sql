-- Migration: Create companies table
-- Companies are deduplicated by (name, postal_code, city)

CREATE TABLE IF NOT EXISTS companies (
    id BIGSERIAL PRIMARY KEY,
    name TEXT NOT NULL,
    street TEXT,
    house_number TEXT,
    postal_code TEXT,
    city TEXT,
    country_iso_code TEXT DEFAULT 'CH',
    phone TEXT,
    email TEXT,
    website TEXT,
    surrogate BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Unique index for company deduplication
CREATE UNIQUE INDEX IF NOT EXISTS idx_companies_unique 
    ON companies(name, COALESCE(postal_code, ''), COALESCE(city, ''));

-- Index for faster lookups
CREATE INDEX IF NOT EXISTS idx_companies_name ON companies(name);
CREATE INDEX IF NOT EXISTS idx_companies_city ON companies(city);

-- Add comment for documentation
COMMENT ON TABLE companies IS 'Normalized companies table - deduplicated by name+postal_code+city';
