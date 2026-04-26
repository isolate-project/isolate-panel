-- WARNING: This migration permanently deletes all traffic statistics data. Cannot be undone.
-- Drop tables
DROP TABLE IF EXISTS traffic_stats;
DROP TABLE IF EXISTS active_connections;
