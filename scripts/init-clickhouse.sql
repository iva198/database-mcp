-- ClickHouse initialization script
-- This script sets up sample data for testing the MCP server

-- Create sample database (if needed)
-- CREATE DATABASE IF NOT EXISTS sample;

-- Create sample tables with different ClickHouse-specific types
CREATE TABLE IF NOT EXISTS events (
    id UInt64,
    user_id UInt32,
    event_type String,
    timestamp DateTime,
    properties Map(String, String),
    tags Array(String),
    metrics Tuple(clicks UInt32, views UInt32, conversions UInt32)
) ENGINE = MergeTree()
ORDER BY (timestamp, user_id)
PARTITION BY toYYYYMM(timestamp);

CREATE TABLE IF NOT EXISTS analytics_summary (
    date Date,
    user_id UInt32,
    total_events UInt64,
    unique_event_types UInt32,
    first_event_time DateTime,
    last_event_time DateTime
) ENGINE = SummingMergeTree()
ORDER BY (date, user_id)
PARTITION BY toYYYYMM(date);

-- Insert sample data
INSERT INTO events (id, user_id, event_type, timestamp, properties, tags, metrics) VALUES
    (1, 101, 'page_view', '2024-01-01 10:00:00', {'page': '/home', 'source': 'direct'}, ['web', 'desktop'], (5, 1, 0)),
    (2, 101, 'click', '2024-01-01 10:05:00', {'element': 'button', 'page': '/home'}, ['web', 'desktop'], (1, 0, 0)),
    (3, 102, 'page_view', '2024-01-01 11:00:00', {'page': '/products', 'source': 'search'}, ['web', 'mobile'], (3, 1, 0)),
    (4, 102, 'purchase', '2024-01-01 11:30:00', {'product': 'widget', 'amount': '99.99'}, ['web', 'mobile'], (0, 0, 1)),
    (5, 103, 'page_view', '2024-01-01 12:00:00', {'page': '/about', 'source': 'social'}, ['web', 'tablet'], (2, 1, 0));

INSERT INTO analytics_summary (date, user_id, total_events, unique_event_types, first_event_time, last_event_time) VALUES
    ('2024-01-01', 101, 2, 2, '2024-01-01 10:00:00', '2024-01-01 10:05:00'),
    ('2024-01-01', 102, 2, 2, '2024-01-01 11:00:00', '2024-01-01 11:30:00'),
    ('2024-01-01', 103, 1, 1, '2024-01-01 12:00:00', '2024-01-01 12:00:00');

-- Create a materialized view for real-time aggregation
CREATE MATERIALIZED VIEW IF NOT EXISTS events_by_hour_mv
ENGINE = SummingMergeTree()
ORDER BY (hour, event_type)
AS SELECT
    toStartOfHour(timestamp) as hour,
    event_type,
    count() as event_count,
    uniq(user_id) as unique_users
FROM events
GROUP BY hour, event_type;