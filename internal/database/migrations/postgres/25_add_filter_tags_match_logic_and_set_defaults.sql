ALTER TABLE filter
    ADD COLUMN tags_match_logic TEXT;

ALTER TABLE filter
    ADD COLUMN except_tags_match_logic TEXT;

UPDATE filter
SET tags_match_logic = 'ANY'
WHERE tags IS NOT NULL;

UPDATE filter
SET except_tags_match_logic = 'ANY'
WHERE except_tags IS NOT NULL;