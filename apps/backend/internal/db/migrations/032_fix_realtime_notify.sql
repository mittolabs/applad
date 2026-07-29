-- Fix realtime table/database channels.
--
-- 001_init.sql historically defined applad_notify_change() twice. The second,
-- later definition won and dropped `database_id` from the NOTIFY payload while
-- also emitting an uppercased action (INSERT/UPDATE/DELETE). The realtime hub
-- builds the database-scoped channel `databases.{project}.{db}` and the
-- table-scoped channel `databases.{project}.{db}.{table}` from `database_id`,
-- and skips both when it is empty. The net effect: row-change subscriptions
-- (the primary realtime use case) received nothing on any install created from
-- the consolidated migration.
--
-- Redefine the function to the correct payload: include database_id and lower()
-- the action. Idempotent, so it repairs existing databases and is a no-op on
-- fresh ones that already have the corrected 001 definition.
CREATE OR REPLACE FUNCTION applad_notify_change()
RETURNS TRIGGER AS $$
DECLARE
    project_id  TEXT := current_setting('applad.project_id',  true);
    database_id TEXT := current_setting('applad.database_id', true);
    payload     TEXT;
BEGIN
    IF project_id IS NULL OR project_id = '' THEN
        RETURN COALESCE(NEW, OLD);
    END IF;
    payload := json_build_object(
        'project_id',  project_id,
        'database_id', database_id,
        'schema',      TG_TABLE_SCHEMA,
        'table',       TG_TABLE_NAME,
        'action',      lower(TG_OP),
        'old',         CASE WHEN TG_OP = 'DELETE' THEN row_to_json(OLD) ELSE NULL END,
        'new',         CASE WHEN TG_OP <> 'DELETE' THEN row_to_json(NEW) ELSE NULL END,
        'timestamp',   to_char(NOW() AT TIME ZONE 'UTC', 'YYYY-MM-DD"T"HH24:MI:SS"Z"')
    )::text;
    PERFORM pg_notify('applad_changes', payload);
    RETURN COALESCE(NEW, OLD);
END;
$$ LANGUAGE plpgsql;
