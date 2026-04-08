-- Add file_security and expose image_transformations for bucket settings
ALTER TABLE buckets ADD COLUMN file_security TINYINT(1) NOT NULL DEFAULT 0 AFTER antivirus;
