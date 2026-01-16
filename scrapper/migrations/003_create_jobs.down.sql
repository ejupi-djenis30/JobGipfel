-- Rollback: Drop jobs table and enums
DROP TABLE IF EXISTS jobs CASCADE;
DROP TYPE IF EXISTS job_status;
DROP TYPE IF EXISTS job_source;
