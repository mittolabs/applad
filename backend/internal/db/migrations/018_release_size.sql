-- What a deploy produced, so the console can show it.
--
-- Build duration and image size were displayed as "--" because nothing ever
-- recorded them. Duration was known and thrown away; size was never asked for.
ALTER TABLE deploy_releases ADD COLUMN IF NOT EXISTS size_bytes BIGINT NOT NULL DEFAULT 0;
