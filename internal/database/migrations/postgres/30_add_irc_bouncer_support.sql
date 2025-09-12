ALTER TABLE irc_network
    ADD COLUMN use_bouncer BOOLEAN DEFAULT FALSE;

ALTER TABLE irc_network
    ADD COLUMN bouncer_addr TEXT;