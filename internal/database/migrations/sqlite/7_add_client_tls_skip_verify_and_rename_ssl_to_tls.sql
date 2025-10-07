ALTER TABLE "client"
    ADD COLUMN tls_skip_verify BOOLEAN DEFAULT FALSE;

ALTER TABLE "client"
    RENAME COLUMN ssl TO tls;
