-- add tls skip verify column to irc_network
ALTER TABLE irc_network
    ADD COLUMN tls_skip_verify BOOLEAN DEFAULT FALSE;

-- set all existing networks to skip tls verification
UPDATE irc_network
SET
    tls_skip_verify = true;
