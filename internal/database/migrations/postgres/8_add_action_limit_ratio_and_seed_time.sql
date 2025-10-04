ALTER TABLE "action"
    ADD COLUMN limit_ratio REAL DEFAULT 0;

ALTER TABLE "action"
    ADD COLUMN limit_seed_time INTEGER DEFAULT 0;
