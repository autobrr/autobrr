-- keys created before scopes were enforced had full access with empty scopes
UPDATE api_key
SET scopes = ARRAY ['*']
WHERE cardinality(scopes) = 0;

ALTER TABLE api_key
    ADD COLUMN id SERIAL;

ALTER TABLE api_key
    ADD COLUMN key_hash TEXT;

UPDATE api_key
SET key_hash = encode(sha256(convert_to(key, 'UTF8')), 'hex');

ALTER TABLE api_key
    DROP COLUMN key;

ALTER TABLE api_key
    ALTER COLUMN key_hash SET NOT NULL;

ALTER TABLE api_key
    ADD CONSTRAINT api_key_key_hash_key UNIQUE (key_hash);

ALTER TABLE api_key
    ADD PRIMARY KEY (id);
