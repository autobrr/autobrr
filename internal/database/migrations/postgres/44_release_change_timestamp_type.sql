ALTER TABLE "release"
    ALTER COLUMN timestamp TYPE timestamptz USING timestamp AT TIME ZONE 'UTC';