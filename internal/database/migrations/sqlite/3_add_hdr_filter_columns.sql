ALTER TABLE "filter"
    ADD COLUMN match_hdr TEXT [] DEFAULT '{}';

ALTER TABLE "filter"
    ADD COLUMN except_hdr TEXT [] DEFAULT '{}';
