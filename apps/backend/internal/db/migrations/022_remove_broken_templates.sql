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

-- deploy_templates has no is_seed/source column to mark a row's origin, and the
-- broken-URL predicate below (NULL/empty/`/tree/`) would just as happily match a
-- template a user created later. This migration already ran on existing installs
-- (it will not re-run), so the risk is a fresh install: scope the delete to the
-- exact ids seeded by 001_init so a user-created row can never be caught. The
-- broken-URL predicate is kept as well so only the genuinely-unclonable seeds go.
DELETE FROM deploy_templates
 WHERE id IN (
        'tpl_nextjs_starter',
        'tpl_astro_blog',
        'tpl_svelte_starter',
        'tpl_nuxt_starter',
        'tpl_react_vite',
        'tpl_vue_starter',
        'tpl_flutter_web',
        'tpl_static_html',
        'tpl_docker_node',
        'tpl_docker_go',
        'tpl_docker_python',
        'tpl_flutter_mobile',
        'tpl_flutter_desktop',
        'tpl_electron',
        'tpl_tauri'
       )
   AND (
        repo_url IS NULL
     OR repo_url = ''
     OR repo_url LIKE '%/tree/%'
     OR repo_url = 'https://github.com/nuxt/starter'
       );
