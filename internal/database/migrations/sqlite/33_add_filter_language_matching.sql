ALTER TABLE "filter"
    ADD COLUMN match_language TEXT [] DEFAULT '{}';

ALTER TABLE "filter"
    ADD COLUMN except_language TEXT [] DEFAULT '{}';
