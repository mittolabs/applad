-- ---------------------------------------------------------------------------
-- Migration 044: Remove Test.
--
-- Running a project's test suite is a product of its own — an image built from
-- the source, a browser recording flows, JUnit parsed back — and it was the
-- largest package in this backend while being the one thing here that no app
-- ever calls. No SDK exposed it: it existed only for a human clicking in the
-- console. By the platform-primitive test, it never was one.
--
-- Test artifacts on disk under STORAGE_PATH/test-artifacts are left alone; a
-- migration deletes rows, not an operator's files.
-- ---------------------------------------------------------------------------

-- Scheduled suites no longer have anything to fire.
DELETE FROM cron_state WHERE kind = 'test_suite';

-- CASCADE takes cases, artifacts and the rest with their parents, so the order
-- of these statements does not matter.
DROP TABLE IF EXISTS test_artifacts CASCADE;
DROP TABLE IF EXISTS test_cases     CASCADE;
DROP TABLE IF EXISTS test_captures  CASCADE;
DROP TABLE IF EXISTS test_flows     CASCADE;
DROP TABLE IF EXISTS test_runs      CASCADE;
DROP TABLE IF EXISTS tests          CASCADE;
DROP TABLE IF EXISTS test_suites    CASCADE;
DROP TABLE IF EXISTS test_runners   CASCADE;
