UPDATE irc_network
SET server = 'irc.animefriends.moe',
    name   = CASE
                 WHEN name = 'AnimeBytes-IRC' THEN 'AnimeBytes'
                 ELSE name
        END
WHERE server = 'irc.animebytes.tv';