-- 0010_invite_roles: multi-manager support (specs/002-multi-manager-support).
-- - user_role gains 'superadmin': the single platform-owner account created
--   by /auth/setup on a fresh deployment (research R1/R6).
-- - invite_codes gains 'role': the role a NEW registrant receives when the
--   invite is redeemed (research R3). Existing rows default to 'resident'.
--
-- Deliberate: NO backfill of pre-existing accounts to 'superadmin' —
-- existing deployments keep their first account as a plain manager (spec
-- assumption; research R2). The new enum value is therefore never *used* in
-- this transaction (PG forbids consuming a value added in the same
-- transaction); all superadmin writes happen in application code later.

BEGIN;

ALTER TYPE user_role ADD VALUE IF NOT EXISTS 'superadmin';

ALTER TABLE invite_codes
    ADD COLUMN role user_role NOT NULL DEFAULT 'resident';

COMMIT;
