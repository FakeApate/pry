ALTER TABLE scan_findings ADD COLUMN category TEXT NOT NULL DEFAULT '';
ALTER TABLE scan_findings ADD COLUMN interest_score INTEGER NOT NULL DEFAULT 0;
ALTER TABLE scan_findings ADD COLUMN tags TEXT NOT NULL DEFAULT '';

CREATE INDEX idx_findings_category ON scan_findings(scan_id, category);
CREATE INDEX idx_findings_interest ON scan_findings(scan_id, interest_score);
