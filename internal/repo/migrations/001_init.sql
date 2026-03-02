-- docs: one row per uploaded book
CREATE TABLE docs (
    id          TEXT PRIMARY KEY,     -- SHA256 hash of the file
    filename    TEXT NOT NULL,
    title       TEXT,
    author      TEXT,
    format      TEXT NOT NULL CHECK (format IN ('pdf', 'epub')),
    page_count  INTEGER,
    file_size   INTEGER NOT NULL,
    uploaded_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    last_opened DATETIME
);

-- sessions: one row per active login
CREATE TABLE sessions (
    token       TEXT PRIMARY KEY,     -- random string stored in cookie
    created_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    expires_at  DATETIME NOT NULL
);

-- positions: where the user was last reading
CREATE TABLE positions (
    doc_id    TEXT PRIMARY KEY REFERENCES docs(id) ON DELETE CASCADE,
    page      INTEGER NOT NULL DEFAULT 1,
    scroll_y  REAL NOT NULL DEFAULT 0.0,
    chapter   TEXT,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);