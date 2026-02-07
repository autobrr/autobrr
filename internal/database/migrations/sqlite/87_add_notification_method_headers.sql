-- Add method and headers columns to notification table for webhook configuration
ALTER TABLE notification ADD COLUMN method TEXT;
ALTER TABLE notification ADD COLUMN headers TEXT;
