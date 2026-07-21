-- Migration 020: build phases
--
-- A pipeline held one build_cmd, which conflated installing dependencies with
-- building the project. The console collected an install command and dropped
-- it on the floor, and the generated Dockerfile had no install step at all, so
-- naming a build command was enough to break the build: `next build` ran in a
-- tree with no node_modules.
--
-- Phases are separate so that overriding one cannot delete another, and so
-- each becomes its own image layer — dependencies are reinstalled only when
-- the lockfile changes.

ALTER TABLE deploy_pipelines
    ADD COLUMN IF NOT EXISTS install_cmd VARCHAR(512),
    ADD COLUMN IF NOT EXISTS start_cmd   VARCHAR(512);
