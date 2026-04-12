-- Add deploy_target_id to platforms so a platform can reference a linked deployment target.
ALTER TABLE platforms ADD COLUMN IF NOT EXISTS deploy_target_id VARCHAR(36);
