ALTER TABLE release_profile_duplicate ADD COLUMN media_processing BOOLEAN DEFAULT FALSE;

-- Release name and Hash only compared anything when both were on, and a profile without field
-- flags matched every grabbed release. Those shapes now compare the release name.
UPDATE release_profile_duplicate
SET release_name = ((COALESCE(release_name, FALSE) AND COALESCE(hash, FALSE)) OR
                    NOT (COALESCE(title, FALSE) OR
                   COALESCE(sub_title, FALSE) OR
                   COALESCE(year, FALSE) OR
                   COALESCE(month, FALSE) OR
                   COALESCE(day, FALSE) OR
                   COALESCE(source, FALSE) OR
                   COALESCE(resolution, FALSE) OR
                   COALESCE(codec, FALSE) OR
                   COALESCE(container, FALSE) OR
                   COALESCE(dynamic_range, FALSE) OR
                   COALESCE(audio, FALSE) OR
                   COALESCE(release_group, FALSE) OR
                   COALESCE(season, FALSE) OR
                   COALESCE(episode, FALSE) OR
                   COALESCE(website, FALSE) OR
                   COALESCE(proper, FALSE) OR
                   COALESCE(repack, FALSE) OR
                   COALESCE(edition, FALSE) OR
                   COALESCE(hybrid, FALSE) OR
                   COALESCE(language, FALSE)));

-- field profiles always kept REMUX and encodes apart without a flag for it
UPDATE release_profile_duplicate
SET media_processing = NOT release_name;
