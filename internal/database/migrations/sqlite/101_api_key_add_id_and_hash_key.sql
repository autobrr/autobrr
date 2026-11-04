-- sha256_hex is registered on the sqlite driver in sqlite_functions.go
ALTER TABLE api_key RENAME TO api_key_old;

CREATE TABLE api_key
(
    id         INTEGER PRIMARY KEY,
    name       TEXT,
    key_hash   TEXT NOT NULL UNIQUE,
    scopes     TEXT []   DEFAULT '{}' NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- keys created before scopes were enforced had full access with empty scopes
INSERT INTO api_key (name, key_hash, scopes, created_at)
SELECT name,
       sha256_hex(key),
       CASE WHEN scopes = '{}' THEN '{"*"}' ELSE scopes END,
       created_at
FROM api_key_old
ORDER BY created_at;

DROP TABLE api_key_old;
