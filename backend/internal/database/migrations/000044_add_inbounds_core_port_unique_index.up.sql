-- Remove duplicate inbounds keeping the latest one
DELETE FROM inbounds WHERE rowid NOT IN (
    SELECT MAX(rowid) FROM inbounds GROUP BY core_id, port
);
CREATE UNIQUE INDEX IF NOT EXISTS idx_inbounds_core_port ON inbounds(core_id, port);