-- filters can point at a deleted profile because SQLite runs without foreign key enforcement
UPDATE filter
SET release_profile_duplicate_id = NULL
WHERE release_profile_duplicate_id IS NOT NULL
  AND release_profile_duplicate_id NOT IN (SELECT id FROM release_profile_duplicate);
