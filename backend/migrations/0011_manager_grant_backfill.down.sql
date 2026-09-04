-- 0011_manager_grant_backfill (down): intentional no-op.
--
-- The up migration heals zero-time granted_at rows with the building's
-- created_at; the original zero timestamps are unrecoverable, so there is
-- nothing to restore. The autoCreateTime model fix stays in application
-- code regardless.

BEGIN;

COMMIT;
