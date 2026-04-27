CREATE TABLE scans (
    scan_id        TEXT PRIMARY KEY,
    url            TEXT NOT NULL,
    status         TEXT NOT NULL DEFAULT 'PENDING' CHECK(status IN ('PENDING', 'RUNNING', 'DONE', 'FAILED')),
    created_at     TEXT NOT NULL DEFAULT (datetime('now')),
    updated_at     TEXT NOT NULL DEFAULT (datetime('now')),
    completed_at   TEXT,
    result         TEXT,
    failure_reason TEXT
);

CREATE INDEX idx_scans_status ON scans(status);
CREATE INDEX idx_scans_created_at ON scans(created_at);

CREATE TABLE scan_findings (
    id             INTEGER PRIMARY KEY,
    scan_id        TEXT NOT NULL REFERENCES scans(scan_id) ON DELETE CASCADE,
    url            TEXT NOT NULL,
    scan_time      TEXT NOT NULL DEFAULT '',
    content_type   TEXT NOT NULL DEFAULT '',
    content_length INTEGER NOT NULL DEFAULT 0,
    last_modified  TEXT,
    category       TEXT NOT NULL DEFAULT '',
    interest_score INTEGER NOT NULL DEFAULT 0,
    tags           TEXT NOT NULL DEFAULT ''
);

CREATE INDEX idx_findings_scan_id ON scan_findings(scan_id);
CREATE INDEX idx_findings_content_type ON scan_findings(scan_id, content_type);
CREATE INDEX idx_findings_size ON scan_findings(scan_id, content_length);
CREATE INDEX idx_findings_category ON scan_findings(scan_id, category);
CREATE INDEX idx_findings_interest ON scan_findings(scan_id, interest_score);
