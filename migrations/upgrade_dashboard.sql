CREATE TABLE IF NOT EXISTS antigravity_insights (
    id SERIAL PRIMARY KEY,
    metric_id VARCHAR(255) NOT NULL,
    is_anomaly BOOLEAN NOT NULL DEFAULT false,
    anomaly_score FLOAT NOT NULL,
    expected_value FLOAT NOT NULL,
    actual_value FLOAT NOT NULL,
    predicted_future FLOAT NOT NULL,
    severity INT NOT NULL DEFAULT 0,
    timestamp TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_antigravity_insights_metric_id ON antigravity_insights(metric_id);
CREATE INDEX idx_antigravity_insights_timestamp ON antigravity_insights(timestamp);

-- Chaos Engineering Log Table
CREATE TABLE IF NOT EXISTS chaos_events (
    id SERIAL PRIMARY KEY,
    target_service VARCHAR(255) NOT NULL,
    blast_radius INT NOT NULL,
    simulated_recovery_time VARCHAR(50),
    initiated_by VARCHAR(100),
    timestamp TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
