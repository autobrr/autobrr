ALTER TABLE filter
    ADD COLUMN match_record_labels TEXT DEFAULT '';

ALTER TABLE filter
    ADD COLUMN except_record_labels TEXT DEFAULT '';