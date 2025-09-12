ALTER TABLE "filter"
    ADD COLUMN match_other TEXT [] DEFAULT '{}';

ALTER TABLE "filter"
    ADD COLUMN except_other TEXT [] DEFAULT '{}';