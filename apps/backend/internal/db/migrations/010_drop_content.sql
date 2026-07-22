-- The standalone CMS is gone: content is now a mode on Databases tables
-- (see 009_content_mode.sql). These tables held the old parallel storage and
-- are dropped here. Order matters: versions and entries reference types.

DROP TABLE IF EXISTS content_versions;
DROP TABLE IF EXISTS content_entries;
DROP TABLE IF EXISTS content_types;
