-- Drop routing_rules table
DROP INDEX IF EXISTS idx_routing_rules_priority;
DROP INDEX IF EXISTS idx_routing_rules_core_id;
DROP TABLE IF EXISTS routing_rules;
