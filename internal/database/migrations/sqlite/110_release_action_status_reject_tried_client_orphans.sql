-- before the tried-client check moved ahead of the insert, an action skipped because its client
-- already rejected the release left its PENDING row behind; resolve those before startup marks
-- every remaining PENDING row as interrupted
UPDATE release_action_status
SET status = 'PUSH_REJECTED', rejections = '{"action client already tried for this release"}'
WHERE status = 'PENDING'
  AND EXISTS (SELECT 1 FROM release_action_status o
              WHERE o.release_id = release_action_status.release_id
                AND o.type = release_action_status.type
                AND COALESCE(o.client, '') = COALESCE(release_action_status.client, '')
                AND o.status = 'PUSH_REJECTED');
