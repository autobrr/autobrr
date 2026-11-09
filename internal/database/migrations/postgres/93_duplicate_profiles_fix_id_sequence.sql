-- the seeded profiles 1-3 were inserted with explicit ids, so the sequence still handed out 1-3
SELECT setval(pg_get_serial_sequence('release_profile_duplicate', 'id'),
              GREATEST((SELECT MAX(id) FROM release_profile_duplicate),
                       (SELECT last_value FROM release_profile_duplicate_id_seq)));
