CREATE TABLE filter_indexer_dg_tmp
(
    filter_id  INTEGER
        CONSTRAINT filter_indexer_filter_id_fkey
            REFERENCES filter,
    indexer_id INTEGER
        CONSTRAINT filter_indexer_indexer_id_fkey
            REFERENCES indexer
            ON DELETE CASCADE,
    PRIMARY KEY (filter_id, indexer_id)
);

INSERT INTO filter_indexer_dg_tmp(filter_id, indexer_id)
SELECT filter_id, indexer_id
FROM filter_indexer;

DROP TABLE filter_indexer;

ALTER TABLE filter_indexer_dg_tmp
    RENAME TO filter_indexer;
