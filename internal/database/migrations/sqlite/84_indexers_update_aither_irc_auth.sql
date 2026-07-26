UPDATE irc_network
SET
    auth_mechanism = 'SASL_PLAIN',
    auth_account = nick,
    auth_password = pass,
    pass = NULL,
    updated_at = CURRENT_TIMESTAMP
WHERE
    server = 'irc.aither.cc'
  AND pass IS NOT NULL
  AND pass != ''
    AND (auth_mechanism IS NULL OR auth_mechanism = '' OR auth_mechanism = 'NONE')
    AND (auth_account IS NULL OR auth_account = '')
    AND (auth_password IS NULL OR auth_password = '');
