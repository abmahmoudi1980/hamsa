-- 0009_password_auth: replace SMS OTP login with phone+password login.
-- - users.password_hash: bcrypt hash (NULL until the user sets a password).
-- - invite_codes: one-time manager-issued codes; redeem registers the user
--   or resets their password. Codes are stored hashed, never plaintext.
-- - otp_codes: dropped — SMS OTP is removed from the product.

BEGIN;

ALTER TABLE users ADD COLUMN password_hash VARCHAR(255);

CREATE TABLE invite_codes (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    phone       VARCHAR(11) NOT NULL,
    code_hash   VARCHAR(64) NOT NULL,
    created_by  UUID REFERENCES users (id),
    expires_at  TIMESTAMPTZ NOT NULL,
    consumed_at TIMESTAMPTZ,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_invite_codes_phone_created ON invite_codes (phone, created_at DESC);

DROP TABLE otp_codes;

COMMIT;
