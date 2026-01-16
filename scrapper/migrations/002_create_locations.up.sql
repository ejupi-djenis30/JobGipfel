-- Migration: Create locations table
-- Locations are deduplicated by (postal_code, city, canton_code)

CREATE TABLE IF NOT EXISTS locations (
    id BIGSERIAL PRIMARY KEY,
    remarks TEXT,
    city TEXT,
    postal_code TEXT,
    communal_code TEXT,
    region_code TEXT,
    canton_code TEXT,
    country_iso_code TEXT DEFAULT 'CH',
    lat DECIMAL(10, 8),
    lon DECIMAL(11, 8),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Unique index for location deduplication
CREATE UNIQUE INDEX IF NOT EXISTS idx_locations_unique 
    ON locations(COALESCE(postal_code, ''), COALESCE(city, ''), COALESCE(canton_code, ''));

-- Indexes for common queries
CREATE INDEX IF NOT EXISTS idx_locations_canton ON locations(canton_code);
CREATE INDEX IF NOT EXISTS idx_locations_city ON locations(city);
CREATE INDEX IF NOT EXISTS idx_locations_postal_code ON locations(postal_code);

-- Add comment for documentation
COMMENT ON TABLE locations IS 'Normalized locations table - deduplicated by postal_code+city+canton_code';
