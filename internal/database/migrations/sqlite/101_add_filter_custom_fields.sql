ALTER TABLE filter ADD COLUMN custom_fields TEXT NOT NULL DEFAULT '[]';
ALTER TABLE filter ADD COLUMN custom_fields_match_logic TEXT NOT NULL DEFAULT 'ALL';
