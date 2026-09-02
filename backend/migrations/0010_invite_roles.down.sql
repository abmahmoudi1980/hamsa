-- 0010_invite_roles (down): remove the invite role column.
--
-- The 'superadmin' value added to user_role is intentionally NOT removed:
-- PostgreSQL cannot drop enum values in place (a full type rebuild would
-- rewrite users/invite_codes), and a stale enum label is harmless — no row
-- references it once the application stops writing it (research R2).

BEGIN;

ALTER TABLE invite_codes DROP COLUMN role;

COMMIT;
