-- Initial database setup for hackathon template
-- This script runs when the PostgreSQL container starts

-- Create extensions
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- Create indexes for better performance
-- Note: GORM will create the tables, but we can add additional indexes here

-- Add any initial data or configurations
-- For example, create an admin user (password: admin123)
-- INSERT INTO users (email, password, first_name, last_name, role, is_active, created_at, updated_at)
-- VALUES (
--     'admin@example.com', 
--     '$2a$12$LQv3c1yqBWVHxkd0LHAkCOYz6TtxMQJqhN8/LeGgAW8/7DAXQMYky', -- admin123
--     'Admin',
--     'User',
--     'admin',
--     true,
--     NOW(),
--     NOW()
-- );

-- Log successful initialization
DO $$
BEGIN
    RAISE NOTICE 'Database initialized successfully for hackathon template';
END $$; 