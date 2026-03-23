-- Drop traffic_stats_hourly table
DROP INDEX IF EXISTS idx_traffic_stats_hourly_timestamp;
DROP INDEX IF EXISTS idx_traffic_stats_hourly_user_id;
DROP TABLE IF EXISTS traffic_stats_hourly;
