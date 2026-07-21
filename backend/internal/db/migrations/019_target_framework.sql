-- What a site is built with, as detected when it is built.
--
-- The console showed every site as "Static" because nothing recorded what
-- detection had already worked out.
ALTER TABLE deploy_targets ADD COLUMN IF NOT EXISTS framework VARCHAR(64);
