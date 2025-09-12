ALTER TABLE indexer
    ADD COLUMN identifier_external TEXT;

UPDATE indexer
SET identifier_external = name;