-- runqy database schema initialization
-- Run this script to create the required tables:
--   psql -h localhost -U postgres -d runqy-dev -f init.sql

-- Queue workers configuration table
CREATE TABLE IF NOT EXISTS queue_workers_config (
    id SERIAL PRIMARY KEY,
    name TEXT UNIQUE NOT NULL,
    priority INTEGER,
    deployment JSONB
);

-- Index for faster prefix lookups (used by ListByPrefix)
CREATE INDEX IF NOT EXISTS idx_queue_workers_config_name ON queue_workers_config(name);
