-- 0001_core: identity & platform tables — users, otp_codes, refresh_tokens,
-- audit_logs (append-only), notifications, files.
-- Per specs/001-building-management-mvp/data-model.md.

BEGIN;

CREATE TYPE user_role AS ENUM ('manager', 'resident');

CREATE TYPE notification_type AS ENUM (
    'invoice_issued',
    'due_soon',
    'overdue',
    'payment_recorded',
    'request_submitted',
    'request_status_changed',
    'announcement_published'
);

-- users: authentication identity (mobile + OTP login).
CREATE TABLE users (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    phone       VARCHAR(11) NOT NULL UNIQUE,
    role        user_role   NOT NULL DEFAULT 'resident',
    name        VARCHAR(120) NOT NULL DEFAULT '',
    fcm_token   VARCHAR(255),
    is_active   BOOLEAN      NOT NULL DEFAULT TRUE,
    created_at  TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ  NOT NULL DEFAULT now(),
    deleted_at  TIMESTAMPTZ
);

-- otp_codes: hashed one-time codes (never plaintext).
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

-- refresh_tokens: rotating opaque tokens (hashed); family revocation on reuse.
CREATE TABLE refresh_tokens (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id     UUID NOT NULL REFERENCES users (id),
    family_id   UUID NOT NULL,
    token_hash  VARCHAR(64) NOT NULL UNIQUE,
    expires_at  TIMESTAMPTZ NOT NULL,
    revoked_at  TIMESTAMPTZ,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_refresh_tokens_user    ON refresh_tokens (user_id);
CREATE INDEX idx_refresh_tokens_family  ON refresh_tokens (family_id);

-- audit_logs: append-only (spec §20 / FR-038).
CREATE TABLE audit_logs (
    id           BIGSERIAL PRIMARY KEY,
    user_id      UUID,
    action       VARCHAR(60)  NOT NULL,
    object_type  VARCHAR(40)  NOT NULL,
    object_id    UUID,
    before_value JSONB,
    after_value  JSONB,
    created_at   TIMESTAMPTZ  NOT NULL DEFAULT now()
);

CREATE INDEX idx_audit_logs_object ON audit_logs (object_type, object_id);
CREATE INDEX idx_audit_logs_user   ON audit_logs (user_id);
CREATE INDEX idx_audit_logs_created ON audit_logs (created_at DESC);

-- Enforce append-only audit trail at the database level.
CREATE OR REPLACE FUNCTION audit_logs_append_only() RETURNS trigger AS $$
BEGIN
    RAISE EXCEPTION 'audit_logs is append-only';
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trg_audit_logs_append_only
    BEFORE UPDATE OR DELETE ON audit_logs
    FOR EACH ROW EXECUTE FUNCTION audit_logs_append_only();

-- notifications: in-app guaranteed channel (FR-032/FR-033).
CREATE TABLE notifications (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id    UUID NOT NULL REFERENCES users (id),
    type       notification_type NOT NULL,
    title      VARCHAR(200) NOT NULL,
    body       TEXT,
    ref_type   VARCHAR(30),
    ref_id     UUID,
    is_read    BOOLEAN     NOT NULL DEFAULT FALSE,
    read_at    TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_notifications_user_read ON notifications (user_id, is_read, created_at DESC);

-- files: uploaded attachments (receipts, photos, announcement attachments).
CREATE TABLE files (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    path         VARCHAR(500) NOT NULL,
    content_type VARCHAR(100) NOT NULL,
    size_bytes   BIGINT       NOT NULL,
    uploaded_by  UUID,
    created_at   TIMESTAMPTZ  NOT NULL DEFAULT now()
);

COMMIT;
