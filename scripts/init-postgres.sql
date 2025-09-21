-- PostgreSQL initialization script
-- This script sets up sample data for testing the MCP server

-- Enable PostGIS extension
CREATE EXTENSION IF NOT EXISTS postgis;

-- Create a sample schema
CREATE SCHEMA IF NOT EXISTS sample;

-- Create sample tables
CREATE TABLE IF NOT EXISTS sample.users (
    id SERIAL PRIMARY KEY,
    username VARCHAR(50) UNIQUE NOT NULL,
    email VARCHAR(100) UNIQUE NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    is_active BOOLEAN DEFAULT true,
    metadata JSONB DEFAULT '{}'::jsonb
);

CREATE TABLE IF NOT EXISTS sample.orders (
    id SERIAL PRIMARY KEY,
    user_id INTEGER REFERENCES sample.users(id),
    total_amount DECIMAL(10,2) NOT NULL,
    status VARCHAR(20) DEFAULT 'pending',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Sample table with PostGIS geometry
CREATE TABLE IF NOT EXISTS sample.locations (
    id SERIAL PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    description TEXT,
    location GEOMETRY(POINT, 4326),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Insert sample data
INSERT INTO sample.users (username, email, metadata) VALUES
    ('alice', 'alice@example.com', '{"role": "admin", "preferences": {"theme": "dark"}}'),
    ('bob', 'bob@example.com', '{"role": "user", "preferences": {"theme": "light"}}'),
    ('charlie', 'charlie@example.com', '{"role": "user", "preferences": {"theme": "auto"}}')
ON CONFLICT (username) DO NOTHING;

INSERT INTO sample.orders (user_id, total_amount, status) VALUES
    (1, 99.99, 'completed'),
    (1, 149.50, 'pending'),
    (2, 29.99, 'completed'),
    (2, 199.99, 'shipped'),
    (3, 79.99, 'pending')
ON CONFLICT DO NOTHING;

INSERT INTO sample.locations (name, description, location) VALUES
    ('San Francisco', 'Golden Gate Bridge area', ST_GeomFromText('POINT(-122.4194 37.7749)', 4326)),
    ('New York', 'Times Square area', ST_GeomFromText('POINT(-73.9857 40.7484)', 4326)),
    ('London', 'Big Ben area', ST_GeomFromText('POINT(-0.1276 51.5007)', 4326))
ON CONFLICT DO NOTHING;

-- Create indexes for better performance
CREATE INDEX IF NOT EXISTS idx_users_email ON sample.users(email);
CREATE INDEX IF NOT EXISTS idx_orders_user_id ON sample.orders(user_id);
CREATE INDEX IF NOT EXISTS idx_orders_status ON sample.orders(status);
CREATE INDEX IF NOT EXISTS idx_locations_geom ON sample.locations USING GIST(location);

-- Create a view for testing
CREATE OR REPLACE VIEW sample.user_order_summary AS
SELECT 
    u.id,
    u.username,
    u.email,
    COUNT(o.id) as total_orders,
    COALESCE(SUM(o.total_amount), 0) as total_spent,
    MAX(o.created_at) as last_order_date
FROM sample.users u
LEFT JOIN sample.orders o ON u.id = o.user_id
GROUP BY u.id, u.username, u.email;

-- Grant permissions (for demonstration)
GRANT USAGE ON SCHEMA sample TO postgres;
GRANT SELECT ON ALL TABLES IN SCHEMA sample TO postgres;
GRANT SELECT ON ALL SEQUENCES IN SCHEMA sample TO postgres;