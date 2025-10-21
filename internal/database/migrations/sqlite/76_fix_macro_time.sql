-- Update macro with typo
UPDATE filter_external
SET
    exec_cmd = REPLACE(REPLACE(exec_cmd, '{{ .CurrenTimeUnixMS }}', '{{ .CurrentTimeUnixMS }}'), '{{.CurrenTimeUnixMS}}', '{{ .CurrentTimeUnixMS }}'),
    exec_args = REPLACE(REPLACE(exec_args, '{{ .CurrenTimeUnixMS }}', '{{ .CurrentTimeUnixMS }}'), '{{.CurrenTimeUnixMS}}', '{{ .CurrentTimeUnixMS }}'),
    webhook_data = REPLACE(REPLACE(webhook_data, '{{ .CurrenTimeUnixMS }}', '{{ .CurrentTimeUnixMS }}'), '{{.CurrenTimeUnixMS}}', '{{ .CurrentTimeUnixMS }}')
WHERE
    exec_cmd LIKE '%{{ .CurrenTimeUnixMS }}%' OR exec_cmd LIKE '%{{.CurrenTimeUnixMS}}%'
   OR exec_args LIKE '%{{ .CurrenTimeUnixMS }}%' OR exec_args LIKE '%{{.CurrenTimeUnixMS}}%'
   OR webhook_data LIKE '%{{ .CurrenTimeUnixMS }}%' OR webhook_data LIKE '%{{.CurrenTimeUnixMS}}%';

UPDATE action
SET
    exec_cmd = REPLACE(REPLACE(exec_cmd, '{{ .CurrenTimeUnixMS }}', '{{ .CurrentTimeUnixMS }}'), '{{.CurrenTimeUnixMS}}', '{{ .CurrentTimeUnixMS }}'),
    exec_args = REPLACE(REPLACE(exec_args, '{{ .CurrenTimeUnixMS }}', '{{ .CurrentTimeUnixMS }}'), '{{.CurrenTimeUnixMS}}', '{{ .CurrentTimeUnixMS }}'),
    watch_folder = REPLACE(REPLACE(watch_folder, '{{ .CurrenTimeUnixMS }}', '{{ .CurrentTimeUnixMS }}'), '{{.CurrenTimeUnixMS}}', '{{ .CurrentTimeUnixMS }}'),
    category = REPLACE(REPLACE(category, '{{ .CurrenTimeUnixMS }}', '{{ .CurrentTimeUnixMS }}'), '{{.CurrenTimeUnixMS}}', '{{ .CurrentTimeUnixMS }}'),
    tags = REPLACE(REPLACE(tags, '{{ .CurrenTimeUnixMS }}', '{{ .CurrentTimeUnixMS }}'), '{{.CurrenTimeUnixMS}}', '{{ .CurrentTimeUnixMS }}'),
    label = REPLACE(REPLACE(label, '{{ .CurrenTimeUnixMS }}', '{{ .CurrentTimeUnixMS }}'), '{{.CurrenTimeUnixMS}}', '{{ .CurrentTimeUnixMS }}'),
    save_path = REPLACE(REPLACE(save_path, '{{ .CurrenTimeUnixMS }}', '{{ .CurrentTimeUnixMS }}'), '{{.CurrenTimeUnixMS}}', '{{ .CurrentTimeUnixMS }}'),
    webhook_data = REPLACE(REPLACE(webhook_data, '{{ .CurrenTimeUnixMS }}', '{{ .CurrentTimeUnixMS }}'), '{{.CurrenTimeUnixMS}}', '{{ .CurrentTimeUnixMS }}')
WHERE
    exec_cmd LIKE '%{{ .CurrenTimeUnixMS }}%' OR exec_cmd LIKE '%{{.CurrenTimeUnixMS}}%'
   OR exec_args LIKE '%{{ .CurrenTimeUnixMS }}%' OR exec_args LIKE '%{{.CurrenTimeUnixMS}}%'
   OR watch_folder LIKE '%{{ .CurrenTimeUnixMS }}%' OR watch_folder LIKE '%{{.CurrenTimeUnixMS}}%'
   OR category LIKE '%{{ .CurrenTimeUnixMS }}%' OR category LIKE '%{{.CurrenTimeUnixMS}}%'
   OR tags LIKE '%{{ .CurrenTimeUnixMS }}%' OR tags LIKE '%{{.CurrenTimeUnixMS}}%'
   OR label LIKE '%{{ .CurrenTimeUnixMS }}%' OR label LIKE '%{{.CurrenTimeUnixMS}}%'
   OR save_path LIKE '%{{ .CurrenTimeUnixMS }}%' OR save_path LIKE '%{{.CurrenTimeUnixMS}}%'
   OR webhook_data LIKE '%{{ .CurrenTimeUnixMS }}%' OR webhook_data LIKE '%{{.CurrenTimeUnixMS}}%';
