-- Migration 022: remove templates that cannot deploy
--
-- Every seeded template was broken. Five carried no repository at all, so
-- there was nothing to clone. Two pointed at github.com/<org>/<repo>/tree/...
-- URLs, which are pages for browsing a subdirectory, not repositories git can
-- clone. The last, nuxt/starter, is clonable but its default branch is
-- "templates" rather than the "main" recorded here, and it is a collection of
-- starters rather than an application.
--
-- A gallery of eight things that all fail is worse than an empty one: it
-- invites somebody to pick, wait for a build, and read a git error. They are
-- removed rather than hidden, so the catalogue can be repopulated with
-- repositories that have been checked.

DELETE FROM deploy_templates
 WHERE repo_url IS NULL
    OR repo_url = ''
    OR repo_url LIKE '%/tree/%'
    OR repo_url = 'https://github.com/nuxt/starter';
