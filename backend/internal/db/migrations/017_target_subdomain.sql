-- A deployed app's subdomain must belong to exactly one target.
--
-- The subdomain was derived from the target's name at deploy time and never
-- stored, so two targets named the same thing resolved to one address and the
-- second silently took over the first. Across projects that is worse than
-- confusing: anyone could claim a subdomain another project was already
-- serving from, simply by naming a target after it.
ALTER TABLE deploy_targets ADD COLUMN IF NOT EXISTS subdomain VARCHAR(128);

-- Backfill from the name, using the same rules the executor applied.
UPDATE deploy_targets
   SET subdomain = trim(both '-' from regexp_replace(lower(name), '[^a-z0-9]+', '-', 'g'))
 WHERE subdomain IS NULL AND type = 'web';

-- Existing duplicates keep the oldest claim; the rest are suffixed so the
-- constraint can be applied without deleting anybody's work.
WITH ranked AS (
    SELECT id, subdomain,
           ROW_NUMBER() OVER (PARTITION BY subdomain ORDER BY created_at) AS rn
      FROM deploy_targets
     WHERE subdomain IS NOT NULL AND subdomain <> ''
)
UPDATE deploy_targets t
   SET subdomain = t.subdomain || '-' || r.rn
  FROM ranked r
 WHERE t.id = r.id AND r.rn > 1;

CREATE UNIQUE INDEX IF NOT EXISTS idx_deploy_targets_subdomain
    ON deploy_targets (subdomain) WHERE subdomain IS NOT NULL AND subdomain <> '';
