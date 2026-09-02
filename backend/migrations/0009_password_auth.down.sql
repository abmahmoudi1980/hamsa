-- 0009_password_auth down: restore otp_codes, drop password auth tables.

BEGIN;

CREATE TABLE otp_codes (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    phone       VARCHAR(11) NOT NULL,
    code_hash   VARCHAR(64) NOT NULL,
    expires_at  TIMESTAMPTZ NOT NULL,
    attempts    INT         NOT NULL DEFAULT 0,
    consumed_at TIMESTAMPTZ,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_otp_codes_phone_created ON otp_codes (phone, created_at DESC);

DROP TABLE invite_codes;

ALTER TABLE users DROP COLUMN password_hash;

COMMIT;
