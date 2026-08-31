-- 0008_announcements: US8 — announcements with audience targeting and read tracking.
-- Per specs/001-building-management-mvp/data-model.md.
BEGIN;

CREATE TYPE announcement_audience AS ENUM ('all', 'block', 'floor', 'unit');

CREATE TABLE announcements (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    building_id     UUID NOT NULL REFERENCES buildings (id),
    title           VARCHAR(200) NOT NULL,
    body            TEXT NOT NULL,
    audience_type   announcement_audience NOT NULL,
    audience_value  VARCHAR(500),
    publish_at      TIMESTAMPTZ,
    expire_at       TIMESTAMPTZ,
    attachment_file VARCHAR(500),
    created_by      UUID REFERENCES users (id),
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT chk_ann_audience CHECK (
        (audience_type = 'all' AND (audience_value IS NULL OR audience_value = '')) OR
        (audience_type IN ('block','floor','unit') AND audience_value IS NOT NULL AND audience_value <> '')
    ),
    CONSTRAINT chk_ann_expire CHECK (expire_at IS NULL OR publish_at IS NULL OR expire_at > publish_at)
);

CREATE INDEX idx_ann_building ON announcements (building_id);
CREATE INDEX idx_ann_building_created ON announcements (building_id, created_at DESC);
CREATE INDEX idx_ann_publish_window ON announcements (publish_at, expire_at);

CREATE TABLE announcement_reads (
    announcement_id UUID NOT NULL REFERENCES announcements (id) ON DELETE CASCADE,
    user_id         UUID NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    read_at         TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (announcement_id, user_id)
);

CREATE INDEX idx_ann_reads_user ON announcement_reads (user_id);

COMMIT;
