-- Apply explicitly to a layout-28 Studio database with writers stopped.
-- Historical jobs keep NULL and retain their original five-minute budget.
BEGIN;
ALTER TABLE component_warmup_runs ADD COLUMN deadline_at DATETIME NULL;
UPDATE schema_version SET version=29;
COMMIT;
