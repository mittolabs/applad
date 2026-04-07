-- ═══════════════════════════════════════════════════════════════════════════
-- 017: Deploy templates, Git connections, Desktop platform, Environments
-- ═══════════════════════════════════════════════════════════════════════════

-- Deploy templates (pre-built starters for Sites/Containers/Mobile/Desktop)
CREATE TABLE IF NOT EXISTS deploy_templates (
    id           VARCHAR(36)  NOT NULL PRIMARY KEY,
    name         VARCHAR(128) NOT NULL,
    description  TEXT,
    category     VARCHAR(32)  NOT NULL,
    framework    VARCHAR(64),
    use_case     VARCHAR(64),
    repo_url     VARCHAR(512),
    branch       VARCHAR(128) NOT NULL DEFAULT 'main',
    build_cmd    VARCHAR(512),
    output_dir   VARCHAR(256),
    install_cmd  VARCHAR(512),
    env_vars     JSON,
    icon         VARCHAR(64),
    popularity   INT          NOT NULL DEFAULT 0,
    created_at   DATETIME(3)  NOT NULL DEFAULT CURRENT_TIMESTAMP(3)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- Git connections (GitHub/GitLab app installations per project)
CREATE TABLE IF NOT EXISTS git_connections (
    id              VARCHAR(36)  NOT NULL PRIMARY KEY,
    project_id      VARCHAR(36)  NOT NULL,
    provider        VARCHAR(32)  NOT NULL DEFAULT 'github',
    installation_id VARCHAR(128),
    access_token    TEXT,
    refresh_token   TEXT,
    account_name    VARCHAR(128),
    account_type    VARCHAR(32),
    expires_at      DATETIME(3),
    created_at      DATETIME(3)  NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    updated_at      DATETIME(3)  NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
    INDEX idx_gc_project (project_id),
    FOREIGN KEY (project_id) REFERENCES projects(id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- Environments (production/staging/development per project)
CREATE TABLE IF NOT EXISTS environments (
    id          VARCHAR(36)  NOT NULL PRIMARY KEY,
    project_id  VARCHAR(36)  NOT NULL,
    name        VARCHAR(64)  NOT NULL,
    slug        VARCHAR(64)  NOT NULL,
    branch      VARCHAR(128),
    domain      VARCHAR(256),
    env_vars    JSON,
    is_default  TINYINT(1)   NOT NULL DEFAULT 0,
    created_at  DATETIME(3)  NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    updated_at  DATETIME(3)  NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
    UNIQUE KEY uk_env (project_id, slug),
    INDEX idx_env_project (project_id),
    FOREIGN KEY (project_id) REFERENCES projects(id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- Link deploy targets to an environment
ALTER TABLE deploy_targets ADD COLUMN IF NOT EXISTS environment_id VARCHAR(36) AFTER project_id;
ALTER TABLE deploy_targets ADD COLUMN IF NOT EXISTS git_connection_id VARCHAR(36) AFTER environment_id;
ALTER TABLE deploy_targets ADD COLUMN IF NOT EXISTS git_repo_url VARCHAR(512) AFTER git_connection_id;
ALTER TABLE deploy_targets ADD COLUMN IF NOT EXISTS git_branch VARCHAR(128) AFTER git_repo_url;

-- ── Seed deploy templates ──

INSERT INTO deploy_templates (id, name, description, category, framework, use_case, repo_url, branch, build_cmd, output_dir, install_cmd, icon) VALUES
-- Sites templates
('tpl_nextjs_starter', 'Next.js Starter', 'A minimal Next.js app with App Router', 'sites', 'nextjs', 'starter', 'https://github.com/vercel/next.js/tree/canary/examples/hello-world', 'main', 'npm run build', '.next', 'npm install', 'nextjs'),
('tpl_astro_blog', 'Astro Blog', 'A blog template built with Astro', 'sites', 'astro', 'blog', 'https://github.com/withastro/astro/tree/main/examples/blog', 'main', 'npm run build', 'dist', 'npm install', 'astro'),
('tpl_svelte_starter', 'SvelteKit Starter', 'A minimal SvelteKit application', 'sites', 'sveltekit', 'starter', 'https://github.com/sveltejs/kit/tree/main/packages/create-svelte/templates/default', 'main', 'npm run build', 'build', 'npm install', 'sveltekit'),
('tpl_nuxt_starter', 'Nuxt Starter', 'A Nuxt 3 starter template', 'sites', 'nuxt', 'starter', 'https://github.com/nuxt/starter', 'main', 'npm run build', '.output/public', 'npm install', 'nuxt'),
('tpl_react_vite', 'React + Vite', 'React with Vite for fast development', 'sites', 'react', 'starter', 'https://github.com/vitejs/vite/tree/main/packages/create-vite/template-react-ts', 'main', 'npm run build', 'dist', 'npm install', 'react'),
('tpl_vue_starter', 'Vue.js Starter', 'Vue 3 with Vite', 'sites', 'vue', 'starter', 'https://github.com/vitejs/vite/tree/main/packages/create-vite/template-vue-ts', 'main', 'npm run build', 'dist', 'npm install', 'vue'),
('tpl_flutter_web', 'Flutter Web', 'Flutter application deployed as a web app', 'sites', 'flutter', 'starter', '', 'main', 'flutter build web --release', 'build/web', '', 'flutter'),
('tpl_static_html', 'Static HTML', 'Plain HTML/CSS/JS site', 'sites', 'static', 'starter', '', 'main', '', '.', '', 'html'),
('tpl_remix_starter', 'Remix Starter', 'Full-stack Remix app', 'sites', 'remix', 'starter', 'https://github.com/remix-run/remix/tree/main/templates/remix', 'main', 'npm run build', 'build', 'npm install', 'remix'),
('tpl_angular_starter', 'Angular Starter', 'Angular with SSR', 'sites', 'angular', 'starter', '', 'main', 'npm run build', 'dist/browser', 'npm install', 'angular'),
('tpl_vitepress', 'VitePress Docs', 'Documentation site with VitePress', 'sites', 'vitepress', 'docs', 'https://github.com/vuejs/vitepress/tree/main/template', 'main', 'npm run docs:build', '.vitepress/dist', 'npm install', 'vitepress'),
('tpl_portfolio', 'Portfolio', 'Personal portfolio template', 'sites', 'astro', 'portfolio', '', 'main', 'npm run build', 'dist', 'npm install', 'portfolio'),
('tpl_ecommerce', 'E-commerce Store', 'Online store template', 'sites', 'nextjs', 'ecommerce', '', 'main', 'npm run build', '.next', 'npm install', 'store'),
('tpl_landing', 'Landing Page', 'Marketing landing page', 'sites', 'react', 'landing', '', 'main', 'npm run build', 'dist', 'npm install', 'landing'),

-- Container templates
('tpl_docker_node', 'Node.js API', 'Node.js REST API with Express', 'containers', 'nodejs', 'api', 'https://github.com/expressjs/express', 'main', '', '', 'npm install', 'nodejs'),
('tpl_docker_go', 'Go API', 'Go REST API with Chi router', 'containers', 'go', 'api', '', 'main', 'go build -o /app/server ./cmd/api', '', '', 'go'),
('tpl_docker_python', 'Python API', 'Python FastAPI application', 'containers', 'python', 'api', '', 'main', '', '', 'pip install -r requirements.txt', 'python'),
('tpl_docker_rust', 'Rust API', 'Rust API with Actix-web', 'containers', 'rust', 'api', '', 'main', 'cargo build --release', '', '', 'rust'),

-- Mobile templates
('tpl_flutter_mobile', 'Flutter Mobile', 'Cross-platform Flutter app', 'mobile', 'flutter', 'starter', '', 'main', 'flutter build apk --release', 'build/app/outputs/flutter-apk', '', 'flutter'),
('tpl_react_native', 'React Native', 'React Native application', 'mobile', 'react-native', 'starter', '', 'main', '', '', 'npm install', 'react'),

-- Desktop templates
('tpl_flutter_desktop', 'Flutter Desktop', 'Cross-platform Flutter desktop app', 'desktop', 'flutter', 'starter', '', 'main', 'flutter build linux --release', 'build/linux/x64/release/bundle', '', 'flutter'),
('tpl_electron', 'Electron App', 'Desktop app with Electron', 'desktop', 'electron', 'starter', '', 'main', 'npm run build', 'dist', 'npm install', 'electron'),
('tpl_tauri', 'Tauri App', 'Lightweight desktop app with Tauri', 'desktop', 'tauri', 'starter', '', 'main', 'npm run tauri build', 'src-tauri/target/release', 'npm install', 'tauri');
