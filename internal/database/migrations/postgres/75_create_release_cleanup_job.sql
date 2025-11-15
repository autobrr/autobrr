-- Enable scheduled cleanup jobs for release history
CREATE TABLE release_cleanup_job
(
    id SERIAL PRIMARY KEY,
    name TEXT NOT NULL,
    enabled BOOLEAN DEFAULT FALSE,
    schedule TEXT NOT NULL,
    older_than INTEGER NOT NULL,
    indexers TEXT,
    statuses TEXT,
    last_run TIMESTAMP,
    last_run_status TEXT,
    last_run_data TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
