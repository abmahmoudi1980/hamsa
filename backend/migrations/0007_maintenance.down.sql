BEGIN;

DROP TABLE IF EXISTS maintenance_requests;
DROP TYPE IF EXISTS maintenance_status;
DROP TYPE IF EXISTS maintenance_priority;
DROP TYPE IF EXISTS maintenance_category;

COMMIT;
