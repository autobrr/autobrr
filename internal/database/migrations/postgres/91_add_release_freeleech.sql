-- Migration 4 dropped these, so upgraded installs lack them, while installs
-- created from the current schema already have them.
ALTER TABLE "release"
    ADD COLUMN IF NOT EXISTS freeleech BOOLEAN DEFAULT FALSE;

ALTER TABLE "release"
    ADD COLUMN IF NOT EXISTS freeleech_percent INTEGER DEFAULT 0;
