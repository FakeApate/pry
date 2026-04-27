# ML-assisted interest scoring (potential improvement)

Status: **deferred** — captured 2026-04-22, not on the active roadmap.

## Goal
Augment the rule-based interest score with a learned per-user signal. While using the TUI, the user marks findings as interesting / not-interesting; once enough labels exist, a model flags files the rules miss, and the user confirms or denies — active-learning loop.

## Shape of the work

Two stages, deliberately split so labels accumulate for months before the model arrives:

### Stage A — Labeling capture (Phase 1.5, small)
Add labeling UI + storage only. No ML, no flagging. Labels survive rescans and outlive scans so training data doesn't evaporate.

**Schema** (new migration `003_finding_labels`):
```sql
CREATE TABLE finding_labels (
    url        TEXT PRIMARY KEY,
    label      TEXT NOT NULL CHECK(label IN ('interesting', 'not_interesting')),
    labeled_at TEXT NOT NULL DEFAULT (datetime('now')),
    updated_at TEXT NOT NULL DEFAULT (datetime('now'))
);
CREATE INDEX idx_finding_labels_label ON finding_labels(label);
```

Three-state: unlabeled = no row. Keyed on full URL (hostname + path) — labels persist across rescans of the same target, don't leak across targets by path alone. No FK to `scan_findings`; labels outlive scans on purpose.

**Storage API** (SQLC, new `internal/store/queries/labels_queries.sql`):
- `UpsertFindingLabel(url, label)`
- `DeleteFindingLabel(url)`
- `GetFindingLabel(url)`
- `CountFindingLabels()` — readiness signal for Stage B
- `QueryFindings` gains `LEFT JOIN finding_labels` so each returned `Finding` carries a `Label *string`

**TUI:**
- New `L` column (2 chars) in the findings table: `★` interesting (accent), `✗` not-interesting (muted), blank unlabeled
- Same marker on leaf nodes in the tree view
- Single keybind **`l`** cycles label on the selected row: unlabeled → interesting → not-interesting → unlabeled. Works in both table and tree view.
- Status bar: `labels: N★ M✗`

**Out of scope for Stage A:**
- Filtering by label
- Labels in HTML/JSON/CSV export (fold into Phase 3)
- Bulk/CLI labeling

**Size:** ~250 lines (migration + 4 SQLC queries + join in `QueryFindings` + column renderer + keybind + tree-view marker + tests).

### Stage B — ML model + active learning (new Phase 6+, after Phase 5)
Prerequisites: Stage A has been in use long enough to accumulate labels (rough target: ≥100 interesting + ≥100 not-interesting before the model is worth training).

**Model:** logistic regression in pure Go. Few hundred lines, trains in ms on thousands of samples, calibrated probabilities 0–1, interpretable per-finding. Rules out CGO (ONNX Runtime) and Python sidecars — both break single-binary distribution.

**Role:** augments, does not replace, the rule-based score. Rules give a cold-start score from finding #1; ML learns personal corrections on top ("this user doesn't care about `.pdf`, always cares about `/config/`").

**Open design decisions** (explicitly deferred to when Stage B is planned):
1. Global model vs per-scan-URL model. Per-scan won't have enough labels unless the same targets are rescanned; global needs features that don't leak target identity (drop hostname, keep path structure).
2. Feature set. Default proposal: one-hot category, top-N extension one-hot, log(size), size bucket, tag flags, existing `interest_score`, hashed path tokens. Skip embeddings — binary-size + training cost.
3. Retrain trigger. Options: every new label (simple, slow), batched every N labels, or on TUI launch (preferred — predictable, non-blocking).

**UI contract** (preliminary):
- Findings row gains an `ml_score` alongside `interest_score`
- When `ml_score` is high but `interest_score` is low (rules missed it), render with a distinctive marker — the "ML flagged this, confirm?" case
- Confirm/deny folds back into `finding_labels` → retrain

## Why deferred
Phase 1.5 alone is cheap (~250 LoC) but adds TUI surface area and a new table while the smart-tagging engine (Phase 1) is still settling. Stage B requires Stage A to have been running long enough to accumulate a training set. Revisit once Phase 1 has stabilized and there's appetite to wire in the labeling UI as a standalone small phase.

## Resolved design decisions
- **Label shape:** three-state (`interesting / not_interesting / unlabeled`), stored as presence/absence of a row with an explicit label column. Negative labels are much more informative than unlabeled-as-negative and changing the schema later once hundreds of labels exist is painful.
- **Label scope:** per full URL (hostname + path). Survives rescans of the same target; doesn't spuriously apply across targets by path alone.
- **Phase split:** labeling and ML are separate phases. Labeling has to land earlier than the model so data can accumulate.
- **Model family:** pure-Go logistic regression, augments (not replaces) the rule-based score.
