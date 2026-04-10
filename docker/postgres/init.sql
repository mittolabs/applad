CREATE ROLE applad_anon NOLOGIN;
CREATE ROLE applad_user NOLOGIN;

GRANT applad_anon TO applad;
GRANT applad_user TO applad;

CREATE SCHEMA IF NOT EXISTS applad;

CREATE OR REPLACE FUNCTION applad.check_jwt() RETURNS void AS $$
DECLARE
    claims jsonb;
    proj_id text;
    usr_id text;
BEGIN
    IF current_setting('request.jwt.claims', true) = '' THEN
        RETURN;
    END IF;

    claims := current_setting('request.jwt.claims', true)::jsonb;
    proj_id := claims->>'project_id';
    usr_id := claims->>'user_id';

    IF proj_id IS NOT NULL AND proj_id <> '' THEN
        PERFORM set_config('applad.project_id', proj_id, true);
    END IF;

    IF usr_id IS NOT NULL AND usr_id <> '' THEN
        PERFORM set_config('applad.user_id', usr_id, true);
    END IF;
END;
$$ LANGUAGE plpgsql;