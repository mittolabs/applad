-- ═══════════════════════════════════════════════════════════════════════════
-- 017: Future features — audit, analytics, search, jobs, billing,
--      vectors, content, regions, edge functions, caching
-- ═══════════════════════════════════════════════════════════════════════════

-- ── Audit logs ───────────────────────────────────────────────────────────────

CREATE TABLE IF NOT EXISTS audit_logs (
    id              VARCHAR(36)   NOT NULL PRIMARY KEY,
    project_id      VARCHAR(36)   NOT NULL,
    user_id         VARCHAR(36),
    action          VARCHAR(128)  NOT NULL,
    resource_type   VARCHAR(64)   NOT NULL,
    resource_id     VARCHAR(36),
    method          VARCHAR(16)   NOT NULL DEFAULT '',
    path            VARCHAR(512)  NOT NULL DEFAULT '',
    status_code     SMALLINT      NOT NULL DEFAULT 0,
    ip_address      VARCHAR(64),
    user_agent      VARCHAR(512),
    metadata        JSON,
    created_at      DATETIME(3)   NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    INDEX idx_al_project    (project_id),
    INDEX idx_al_user       (user_id),
    INDEX idx_al_action     (action),
    INDEX idx_al_resource   (resource_type, resource_id),
    INDEX idx_al_created    (created_at),
    FOREIGN KEY (project_id) REFERENCES projects(id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- ── Analytics ────────────────────────────────────────────────────────────────

CREATE TABLE IF NOT EXISTS analytics_events (
    id          VARCHAR(36)   NOT NULL PRIMARY KEY,
    project_id  VARCHAR(36)   NOT NULL,
    user_id     VARCHAR(36),
    session_id  VARCHAR(36),
    event       VARCHAR(128)  NOT NULL,
    properties  JSON,
    url         VARCHAR(2048),
    referrer    VARCHAR(2048),
    device_type VARCHAR(32),
    browser     VARCHAR(64),
    country     VARCHAR(8),
    created_at  DATETIME(3)   NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    INDEX idx_ae_project  (project_id),
    INDEX idx_ae_user     (user_id),
    INDEX idx_ae_session  (session_id),
    INDEX idx_ae_event    (project_id, event),
    INDEX idx_ae_created  (project_id, created_at),
    FOREIGN KEY (project_id) REFERENCES projects(id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS analytics_sessions (
    id          VARCHAR(36)   NOT NULL PRIMARY KEY,
    project_id  VARCHAR(36)   NOT NULL,
    user_id     VARCHAR(36),
    device_type VARCHAR(32),
    browser     VARCHAR(64),
    country     VARCHAR(8),
    entry_url   VARCHAR(2048),
    exit_url    VARCHAR(2048),
    page_views  INT           NOT NULL DEFAULT 0,
    duration_s  INT           NOT NULL DEFAULT 0,
    started_at  DATETIME(3)   NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    ended_at    DATETIME(3),
    INDEX idx_as_project (project_id),
    INDEX idx_as_user    (user_id),
    INDEX idx_as_started (project_id, started_at),
    FOREIGN KEY (project_id) REFERENCES projects(id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS analytics_funnels (
    id          VARCHAR(36)   NOT NULL PRIMARY KEY,
    project_id  VARCHAR(36)   NOT NULL,
    name        VARCHAR(128)  NOT NULL,
    steps       JSON          NOT NULL,
    created_at  DATETIME(3)   NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    updated_at  DATETIME(3)   NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
    INDEX idx_af_project (project_id),
    FOREIGN KEY (project_id) REFERENCES projects(id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- ── Search ───────────────────────────────────────────────────────────────────

CREATE TABLE IF NOT EXISTS search_indexes (
    id              VARCHAR(36)   NOT NULL PRIMARY KEY,
    project_id      VARCHAR(36)   NOT NULL,
    collection_id   VARCHAR(36),
    name            VARCHAR(128)  NOT NULL,
    fields          JSON          NOT NULL,
    synonyms        JSON,
    ranking_rules   JSON,
    typo_tolerance  TINYINT(1)    NOT NULL DEFAULT 1,
    status          VARCHAR(32)   NOT NULL DEFAULT 'ready',
    created_at      DATETIME(3)   NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    updated_at      DATETIME(3)   NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
    UNIQUE KEY uq_si_project_name (project_id, name),
    INDEX idx_si_project (project_id),
    FOREIGN KEY (project_id) REFERENCES projects(id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS search_synonyms (
    id          VARCHAR(36)  NOT NULL PRIMARY KEY,
    index_id    VARCHAR(36)  NOT NULL,
    synonyms    JSON         NOT NULL,
    created_at  DATETIME(3)  NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    INDEX idx_ss_index (index_id),
    FOREIGN KEY (index_id) REFERENCES search_indexes(id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS search_documents (
    id          VARCHAR(36)    NOT NULL PRIMARY KEY,
    index_id    VARCHAR(36)    NOT NULL,
    project_id  VARCHAR(36)    NOT NULL,
    doc_id      VARCHAR(36)    NOT NULL,
    content     MEDIUMTEXT     NOT NULL,
    metadata    JSON,
    indexed_at  DATETIME(3)    NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    UNIQUE KEY uq_sd_index_doc (index_id, doc_id),
    INDEX idx_sd_index   (index_id),
    FULLTEXT KEY ft_sd_content (content),
    FOREIGN KEY (index_id) REFERENCES search_indexes(id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- ── Scheduled Jobs / Queues ──────────────────────────────────────────────────

CREATE TABLE IF NOT EXISTS job_queues (
    id                  VARCHAR(36)   NOT NULL PRIMARY KEY,
    project_id          VARCHAR(36)   NOT NULL,
    name                VARCHAR(128)  NOT NULL,
    worker_url          VARCHAR(512),
    concurrency         INT           NOT NULL DEFAULT 10,
    retry_limit         INT           NOT NULL DEFAULT 3,
    retry_delay_s       INT           NOT NULL DEFAULT 60,
    dead_letter_queue_id VARCHAR(36),
    paused              TINYINT(1)    NOT NULL DEFAULT 0,
    created_at          DATETIME(3)   NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    updated_at          DATETIME(3)   NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
    UNIQUE KEY uq_jq_project_name (project_id, name),
    INDEX idx_jq_project (project_id),
    FOREIGN KEY (project_id) REFERENCES projects(id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS jobs (
    id              VARCHAR(36)   NOT NULL PRIMARY KEY,
    queue_id        VARCHAR(36)   NOT NULL,
    project_id      VARCHAR(36)   NOT NULL,
    name            VARCHAR(128)  NOT NULL,
    payload         JSON,
    status          VARCHAR(32)   NOT NULL DEFAULT 'pending',
    priority        TINYINT       NOT NULL DEFAULT 0,
    run_at          DATETIME(3)   NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    attempts        INT           NOT NULL DEFAULT 0,
    max_attempts    INT           NOT NULL DEFAULT 3,
    last_error      TEXT,
    depends_on      JSON,
    completed_at    DATETIME(3),
    created_at      DATETIME(3)   NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    updated_at      DATETIME(3)   NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
    INDEX idx_jobs_queue     (queue_id, status, run_at),
    INDEX idx_jobs_project   (project_id),
    INDEX idx_jobs_status    (status, run_at),
    FOREIGN KEY (queue_id)   REFERENCES job_queues(id)  ON DELETE CASCADE,
    FOREIGN KEY (project_id) REFERENCES projects(id)    ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- ── Billing & Metering ───────────────────────────────────────────────────────

CREATE TABLE IF NOT EXISTS billing_plans (
    id              VARCHAR(36)   NOT NULL PRIMARY KEY,
    name            VARCHAR(64)   NOT NULL,
    slug            VARCHAR(64)   NOT NULL UNIQUE,
    price_monthly   INT           NOT NULL DEFAULT 0,
    price_yearly    INT           NOT NULL DEFAULT 0,
    limits          JSON,
    features        JSON,
    active          TINYINT(1)    NOT NULL DEFAULT 1,
    created_at      DATETIME(3)   NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    updated_at      DATETIME(3)   NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS billing_subscriptions (
    id                      VARCHAR(36)   NOT NULL PRIMARY KEY,
    project_id              VARCHAR(36)   NOT NULL UNIQUE,
    plan_id                 VARCHAR(36)   NOT NULL,
    status                  VARCHAR(32)   NOT NULL DEFAULT 'active',
    stripe_customer_id      VARCHAR(128),
    stripe_subscription_id  VARCHAR(128),
    current_period_start    DATETIME(3),
    current_period_end      DATETIME(3),
    cancel_at_period_end    TINYINT(1)    NOT NULL DEFAULT 0,
    created_at              DATETIME(3)   NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    updated_at              DATETIME(3)   NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
    INDEX idx_bs_project (project_id),
    INDEX idx_bs_plan    (plan_id),
    FOREIGN KEY (project_id) REFERENCES projects(id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS billing_events (
    id          VARCHAR(36)   NOT NULL PRIMARY KEY,
    project_id  VARCHAR(36)   NOT NULL,
    event_type  VARCHAR(64)   NOT NULL,
    quantity    BIGINT        NOT NULL DEFAULT 0,
    unit        VARCHAR(32)   NOT NULL DEFAULT '',
    metadata    JSON,
    created_at  DATETIME(3)   NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    INDEX idx_be_project  (project_id),
    INDEX idx_be_type     (project_id, event_type),
    INDEX idx_be_created  (project_id, created_at),
    FOREIGN KEY (project_id) REFERENCES projects(id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS invoices (
    id                  VARCHAR(36)   NOT NULL PRIMARY KEY,
    project_id          VARCHAR(36)   NOT NULL,
    subscription_id     VARCHAR(36),
    amount_cents        INT           NOT NULL DEFAULT 0,
    currency            VARCHAR(8)    NOT NULL DEFAULT 'usd',
    status              VARCHAR(32)   NOT NULL DEFAULT 'draft',
    stripe_invoice_id   VARCHAR(128),
    period_start        DATETIME(3),
    period_end          DATETIME(3),
    paid_at             DATETIME(3),
    created_at          DATETIME(3)   NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    INDEX idx_inv_project (project_id),
    FOREIGN KEY (project_id) REFERENCES projects(id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- ── AI / Vector Service ──────────────────────────────────────────────────────

CREATE TABLE IF NOT EXISTS vector_indexes (
    id              VARCHAR(36)   NOT NULL PRIMARY KEY,
    project_id      VARCHAR(36)   NOT NULL,
    name            VARCHAR(128)  NOT NULL,
    dimensions      INT           NOT NULL DEFAULT 1536,
    metric          VARCHAR(32)   NOT NULL DEFAULT 'cosine',
    collection_id   VARCHAR(36),
    embedding_field VARCHAR(128),
    model           VARCHAR(128)  NOT NULL DEFAULT 'text-embedding-3-small',
    status          VARCHAR(32)   NOT NULL DEFAULT 'ready',
    created_at      DATETIME(3)   NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    updated_at      DATETIME(3)   NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
    UNIQUE KEY uq_vi_project_name (project_id, name),
    INDEX idx_vi_project (project_id),
    FOREIGN KEY (project_id) REFERENCES projects(id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS vector_embeddings (
    id          VARCHAR(36)     NOT NULL PRIMARY KEY,
    index_id    VARCHAR(36)     NOT NULL,
    project_id  VARCHAR(36)     NOT NULL,
    doc_id      VARCHAR(36)     NOT NULL,
    vector      MEDIUMTEXT      NOT NULL,
    metadata    JSON,
    created_at  DATETIME(3)     NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    UNIQUE KEY uq_ve_index_doc (index_id, doc_id),
    INDEX idx_ve_index   (index_id),
    INDEX idx_ve_project (project_id),
    FOREIGN KEY (index_id)   REFERENCES vector_indexes(id) ON DELETE CASCADE,
    FOREIGN KEY (project_id) REFERENCES projects(id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- ── CMS / Content Layer ──────────────────────────────────────────────────────

CREATE TABLE IF NOT EXISTS content_types (
    id              VARCHAR(36)   NOT NULL PRIMARY KEY,
    project_id      VARCHAR(36)   NOT NULL,
    name            VARCHAR(128)  NOT NULL,
    slug            VARCHAR(128)  NOT NULL,
    fields          JSON          NOT NULL,
    versioning      TINYINT(1)    NOT NULL DEFAULT 1,
    localization    TINYINT(1)    NOT NULL DEFAULT 0,
    created_at      DATETIME(3)   NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    updated_at      DATETIME(3)   NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
    UNIQUE KEY uq_ct_project_slug (project_id, slug),
    INDEX idx_ct_project (project_id),
    FOREIGN KEY (project_id) REFERENCES projects(id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS content_entries (
    id          VARCHAR(36)   NOT NULL PRIMARY KEY,
    type_id     VARCHAR(36)   NOT NULL,
    project_id  VARCHAR(36)   NOT NULL,
    slug        VARCHAR(256),
    status      VARCHAR(32)   NOT NULL DEFAULT 'draft',
    locale      VARCHAR(16)   NOT NULL DEFAULT 'en',
    author_id   VARCHAR(36),
    published_at DATETIME(3),
    created_at  DATETIME(3)   NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    updated_at  DATETIME(3)   NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
    INDEX idx_ce_type    (type_id),
    INDEX idx_ce_project (project_id),
    INDEX idx_ce_slug    (project_id, slug),
    INDEX idx_ce_status  (project_id, status),
    FOREIGN KEY (type_id)    REFERENCES content_types(id) ON DELETE CASCADE,
    FOREIGN KEY (project_id) REFERENCES projects(id)      ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS content_versions (
    id          VARCHAR(36)   NOT NULL PRIMARY KEY,
    entry_id    VARCHAR(36)   NOT NULL,
    version     INT           NOT NULL DEFAULT 1,
    data        JSON          NOT NULL,
    created_by  VARCHAR(36),
    created_at  DATETIME(3)   NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    UNIQUE KEY uq_cv_entry_version (entry_id, version),
    INDEX idx_cv_entry (entry_id),
    FOREIGN KEY (entry_id) REFERENCES content_entries(id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- ── Multi-region / Data Residency ────────────────────────────────────────────

CREATE TABLE IF NOT EXISTS regions (
    id          VARCHAR(36)   NOT NULL PRIMARY KEY,
    name        VARCHAR(64)   NOT NULL,
    code        VARCHAR(16)   NOT NULL UNIQUE,
    location    VARCHAR(128)  NOT NULL,
    endpoint    VARCHAR(256)  NOT NULL DEFAULT '',
    latitude    DECIMAL(9,6)  DEFAULT NULL,
    longitude   DECIMAL(9,6)  DEFAULT NULL,
    status      VARCHAR(32)   NOT NULL DEFAULT 'active',
    created_at  DATETIME(3)   NOT NULL DEFAULT CURRENT_TIMESTAMP(3)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS project_regions (
    id              VARCHAR(36)   NOT NULL PRIMARY KEY,
    project_id      VARCHAR(36)   NOT NULL,
    region_id       VARCHAR(36)   NOT NULL,
    primary_region  TINYINT(1)    NOT NULL DEFAULT 0,
    gdpr            TINYINT(1)    NOT NULL DEFAULT 0,
    hipaa           TINYINT(1)    NOT NULL DEFAULT 0,
    created_at      DATETIME(3)   NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    UNIQUE KEY uq_pr_project_region (project_id, region_id),
    INDEX idx_pr_project (project_id),
    FOREIGN KEY (project_id) REFERENCES projects(id) ON DELETE CASCADE,
    FOREIGN KEY (region_id)  REFERENCES regions(id)  ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- Seed default regions
INSERT IGNORE INTO regions (id, name, code, location, endpoint, latitude, longitude) VALUES
    ('region-us-east-1',  'US East (N. Virginia)',   'us-east-1',    'Ashburn, VA, USA',     '', 38.9072,  -77.0369),
    ('region-us-west-2',  'US West (Oregon)',         'us-west-2',    'The Dalles, OR, USA',  '', 45.5946, -121.1787),
    ('region-eu-west-1',  'EU West (Ireland)',        'eu-west-1',    'Dublin, Ireland',      '', 53.3498,  -6.2603),
    ('region-eu-central', 'EU Central (Frankfurt)',   'eu-central-1', 'Frankfurt, Germany',   '', 50.1109,   8.6821),
    ('region-ap-south-1', 'AP South (Mumbai)',        'ap-south-1',   'Mumbai, India',        '', 19.0760,  72.8777),
    ('region-ap-east-1',  'AP East (Singapore)',      'ap-east-1',    'Singapore',            '',  1.3521, 103.8198),
    ('region-sa-east-1',  'SA East (São Paulo)',      'sa-east-1',    'São Paulo, Brazil',    '', -23.5505, -46.6333);

-- ── Edge Functions ───────────────────────────────────────────────────────────

CREATE TABLE IF NOT EXISTS edge_functions (
    id          VARCHAR(36)   NOT NULL PRIMARY KEY,
    project_id  VARCHAR(36)   NOT NULL,
    name        VARCHAR(128)  NOT NULL,
    slug        VARCHAR(128)  NOT NULL,
    code        MEDIUMTEXT    NOT NULL DEFAULT '',
    runtime     VARCHAR(32)   NOT NULL DEFAULT 'js',
    regions     JSON,
    env_vars    JSON,
    status      VARCHAR(32)   NOT NULL DEFAULT 'draft',
    created_at  DATETIME(3)   NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    updated_at  DATETIME(3)   NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
    UNIQUE KEY uq_ef_project_slug (project_id, slug),
    INDEX idx_ef_project (project_id),
    FOREIGN KEY (project_id) REFERENCES projects(id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS edge_deployments (
    id              VARCHAR(36)   NOT NULL PRIMARY KEY,
    function_id     VARCHAR(36)   NOT NULL,
    project_id      VARCHAR(36)   NOT NULL,
    version         INT           NOT NULL DEFAULT 1,
    status          VARCHAR(32)   NOT NULL DEFAULT 'deploying',
    regions         JSON,
    deployed_at     DATETIME(3),
    created_at      DATETIME(3)   NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    INDEX idx_ed_function (function_id),
    INDEX idx_ed_project  (project_id),
    FOREIGN KEY (function_id) REFERENCES edge_functions(id) ON DELETE CASCADE,
    FOREIGN KEY (project_id)  REFERENCES projects(id)       ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
