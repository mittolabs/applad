-- Migration 027: ask answerable questions, not for the abstraction
--
-- "What is the impact of this — low, medium or high?" is the same guess the
-- matrix was meant to remove, asked one level up. Nobody knows what medium
-- impact means, and two people never mean the same thing by it. Everybody can
-- answer "is there a workaround?".
--
-- So each dimension is scored by questions. Options carry scores, the scores
-- sum, and a band turns the sum into a level. The level feeds the grid. Every
-- part is editable because the calibration is a house style, not a fact.
--
-- One option may force the top of the scale on its own — data loss does not
-- need to out-argue three other answers.

CREATE TABLE IF NOT EXISTS plan_priority_questions (
    id         VARCHAR(36)  NOT NULL PRIMARY KEY,
    project_id VARCHAR(36)  NOT NULL,
    -- impact | urgency
    dimension  VARCHAR(16)  NOT NULL,
    text       VARCHAR(512) NOT NULL,
    help       VARCHAR(512) NOT NULL DEFAULT '',
    position   INT          NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    CONSTRAINT fk_q_project FOREIGN KEY (project_id) REFERENCES projects(id) ON DELETE CASCADE
);
CREATE INDEX IF NOT EXISTS idx_q_project ON plan_priority_questions (project_id, dimension, position);

CREATE TABLE IF NOT EXISTS plan_priority_options (
    id          VARCHAR(36)  NOT NULL PRIMARY KEY,
    question_id VARCHAR(36)  NOT NULL,
    label       VARCHAR(256) NOT NULL,
    score       INT          NOT NULL DEFAULT 0,
    -- Set when this answer decides the whole thing by itself.
    forces_top  BOOLEAN      NOT NULL DEFAULT FALSE,
    position    INT          NOT NULL DEFAULT 0,
    CONSTRAINT fk_o_question FOREIGN KEY (question_id) REFERENCES plan_priority_questions(id) ON DELETE CASCADE
);
CREATE INDEX IF NOT EXISTS idx_o_question ON plan_priority_options (question_id, position);

-- What somebody answered for one item.
CREATE TABLE IF NOT EXISTS plan_priority_answers (
    item_id     VARCHAR(36) NOT NULL,
    question_id VARCHAR(36) NOT NULL,
    option_id   VARCHAR(36) NOT NULL,
    answered_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (item_id, question_id),
    CONSTRAINT fk_a_item     FOREIGN KEY (item_id)     REFERENCES plan_items(id) ON DELETE CASCADE,
    CONSTRAINT fk_a_question FOREIGN KEY (question_id) REFERENCES plan_priority_questions(id) ON DELETE CASCADE,
    CONSTRAINT fk_a_option   FOREIGN KEY (option_id)   REFERENCES plan_priority_options(id) ON DELETE CASCADE
);

-- Score to level, per dimension. Three levels because the grid has three.
CREATE TABLE IF NOT EXISTS plan_priority_bands (
    id         VARCHAR(36) NOT NULL PRIMARY KEY,
    project_id VARCHAR(36) NOT NULL,
    dimension  VARCHAR(16) NOT NULL,
    level      SMALLINT    NOT NULL,
    min_score  INT         NOT NULL,
    max_score  INT         NOT NULL,
    CONSTRAINT fk_b_project FOREIGN KEY (project_id) REFERENCES projects(id) ON DELETE CASCADE,
    CONSTRAINT uq_band UNIQUE (project_id, dimension, level)
);
