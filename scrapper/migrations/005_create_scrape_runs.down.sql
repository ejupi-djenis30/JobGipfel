-- Rollback: Drop scrape_runs table
DROP TABLE IF EXISTS scrape_runs CASCADE;
DROP TYPE IF EXISTS scrape_status;
