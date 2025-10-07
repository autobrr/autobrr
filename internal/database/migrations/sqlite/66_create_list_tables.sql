CREATE TABLE list
(
    id                       INTEGER PRIMARY KEY,
    name                     TEXT                   NOT NULL,
    enabled                  BOOLEAN,
    type                     TEXT                   NOT NULL,
    client_id                INTEGER,
    url                      TEXT,
    headers                  TEXT []   DEFAULT '{}' NOT NULL,
    api_key                  TEXT,
    match_release            BOOLEAN,
    tags_included            TEXT []   DEFAULT '{}' NOT NULL,
    tags_excluded            TEXT []   DEFAULT '{}' NOT NULL,
    include_unmonitored      BOOLEAN,
    include_alternate_titles BOOLEAN,
    last_refresh_time        TIMESTAMP,
    last_refresh_status      TEXT,
    last_refresh_data        TEXT,
    created_at               TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at               TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (client_id) REFERENCES client (id) ON DELETE SET NULL
);

CREATE TABLE list_filter
(
    list_id   INTEGER,
    filter_id INTEGER,
    FOREIGN KEY (list_id) REFERENCES list (id) ON DELETE CASCADE,
    FOREIGN KEY (filter_id) REFERENCES filter (id) ON DELETE CASCADE,
    PRIMARY KEY (list_id, filter_id)
);
