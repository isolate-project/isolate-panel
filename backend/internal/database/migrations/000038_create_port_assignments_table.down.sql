-- Drop port_assignments table
DROP INDEX IF EXISTS idx_port_assignments_core;
DROP INDEX IF EXISTS idx_port_assignments_backend;
DROP INDEX IF EXISTS idx_port_assignments_user_port;
DROP TABLE IF EXISTS port_assignments;
