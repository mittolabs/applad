-- Fold the per-recording runners into one.
--
-- Before the catalogue existed, saving a recording created a runner for it
-- alone. That is what made every recording appear twice and what stopped
-- recordings from being grouped or selected. Existing projects are repointed
-- at a single generated runner; the runnable project itself is rebuilt from
-- the flows on the next run, since the flows are the source of truth and the
-- generated source is derived from them.

-- The survivor is the oldest generated runner in each project.
WITH keep AS (
    SELECT DISTINCT ON (project_id) project_id, id
      FROM test_runners
     WHERE source_type = 'generated'
     ORDER BY project_id, created_at
)
UPDATE test_flows f
   SET runner_id = k.id
  FROM keep k
 WHERE f.project_id = k.project_id
   AND f.runner_id IS DISTINCT FROM k.id;

WITH keep AS (
    SELECT DISTINCT ON (project_id) project_id, id
      FROM test_runners
     WHERE source_type = 'generated'
     ORDER BY project_id, created_at
)
DELETE FROM test_runners r
 USING keep k
 WHERE r.source_type = 'generated'
   AND r.project_id = k.project_id
   AND r.id <> k.id;

-- Give the survivor a name that says what it holds.
UPDATE test_runners SET name = 'Recorded flows' WHERE source_type = 'generated';

-- Recorded flows join the catalogue as tests.
INSERT INTO tests (id, project_id, runner_id, suite_name, name, source, flow_id, created_at, updated_at)
SELECT replace(gen_random_uuid()::text, '-', ''), f.project_id, f.runner_id, 'recorded', f.name,
       'recorded', f.id, NOW(), NOW()
  FROM test_flows f
 WHERE f.runner_id IS NOT NULL
ON CONFLICT (project_id, runner_id, suite_name, name) DO NOTHING;

UPDATE test_flows f
   SET test_id = t.id
  FROM tests t
 WHERE t.flow_id = f.id AND f.test_id IS NULL;
