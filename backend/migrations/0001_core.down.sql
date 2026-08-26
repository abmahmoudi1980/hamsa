-- 0001_core (down): drop platform tables in reverse dependency order.

BEGIN;

DROP TABLE IF EXISTS files;
DROP TABLE IF EXISTS notifications;
DROP TRIGGER IF EXISTS trg_audit_logs_append_only ON audit_logs;
DROP FUNCTION IF EXISTS audit_logs_append_only();
DROP TABLE IF EXISTS audit_logs;
DROP TABLE IF EXISTS refresh_tokens;
DROP TABLE IF EXISTS otp_codes;
DROP TABLE IF EXISTS users;
DROP TYPE IF EXISTS notification_type;
DROP TYPE IF EXISTS user_role;

COMMIT;
