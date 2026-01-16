-- Migration: Add normalized fields to job_descriptions
-- These fields store the structured/normalized job content

-- Add normalized content fields (English names)
ALTER TABLE job_descriptions
ADD COLUMN IF NOT EXISTS tasks TEXT,
ADD COLUMN IF NOT EXISTS requirements TEXT,
ADD COLUMN IF NOT EXISTS offer TEXT;

-- Add normalization tracking
ALTER TABLE job_descriptions
ADD COLUMN IF NOT EXISTS is_normalized BOOLEAN NOT NULL DEFAULT FALSE,
ADD COLUMN IF NOT EXISTS normalized_at TIMESTAMPTZ;

-- Add index for finding non-normalized entries
CREATE INDEX IF NOT EXISTS idx_job_descriptions_normalized 
    ON job_descriptions(is_normalized) WHERE is_normalized = FALSE;

-- Add comment for documentation
COMMENT ON COLUMN job_descriptions.tasks IS 'Normalized tasks/duties section extracted by AI';
COMMENT ON COLUMN job_descriptions.requirements IS 'Normalized requirements section extracted by AI';
COMMENT ON COLUMN job_descriptions.offer IS 'Normalized offer/benefits section extracted by AI';
COMMENT ON COLUMN job_descriptions.is_normalized IS 'True if AI has processed this description';
COMMENT ON COLUMN job_descriptions.normalized_at IS 'Timestamp when normalization was performed';
