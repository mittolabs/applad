CREATE TABLE IF NOT EXISTS collection_relationships (
    id                VARCHAR(36)  NOT NULL PRIMARY KEY,
    collection_id     VARCHAR(36)  NOT NULL,
    related_collection VARCHAR(36) NOT NULL,
    relationship_type VARCHAR(32)  NOT NULL,
    two_way           TINYINT(1)   NOT NULL DEFAULT 0,
    `key`             VARCHAR(128) NOT NULL,
    two_way_key       VARCHAR(128),
    on_delete         VARCHAR(32)  NOT NULL DEFAULT 'setNull',
    created_at        DATETIME(3)  NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    INDEX idx_rel_coll (collection_id),
    INDEX idx_rel_related (related_collection),
    FOREIGN KEY (collection_id) REFERENCES collections(id) ON DELETE CASCADE,
    FOREIGN KEY (related_collection) REFERENCES collections(id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
