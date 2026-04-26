-- Add api_port field to cores table for storing random Xray gRPC API port
ALTER TABLE cores ADD COLUMN api_port INTEGER DEFAULT 10085;