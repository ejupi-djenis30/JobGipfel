-- Rollback: Remove normalized fields from job_descriptions

-- Drop index first
DROP INDEX IF EXISTS idx_job_descriptions_normalized;

-- Remove columns
ALTER TABLE job_descriptions
DROP COLUMN IF EXISTS tasks,
DROP COLUMN IF EXISTS requirements,
DROP COLUMN IF EXISTS offer,
DROP COLUMN IF EXISTS is_normalized,
DROP COLUMN IF EXISTS normalized_at;
