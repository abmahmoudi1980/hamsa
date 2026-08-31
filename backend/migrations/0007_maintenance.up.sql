-- 0007_maintenance: US7 — maintenance requests with status workflow
-- (new → under_review → in_progress → done → closed, early close allowed)
-- Per specs/001-building-management-mvp/data-model.md.
BEGIN;

CREATE TYPE maintenance_category AS ENUM (
    'elevator', 'utilities', 'electrical', 'water', 'cleaning',
    'common_area', 'parking', 'other'
);

CREATE TYPE maintenance_priority AS ENUM ('normal', 'important', 'urgent');

CREATE TYPE maintenance_status AS ENUM (
    'new', 'under_review', 'in_progress', 'done', 'closed'
);

CREATE TABLE maintenance_requests (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    building_id         UUID NOT NULL REFERENCES buildings (id),
    unit_id             UUID REFERENCES units (id),
    submitted_by        UUID NOT NULL REFERENCES users (id),
    title               VARCHAR(150) NOT NULL,
    category            maintenance_category NOT NULL,
    description         TEXT,
    location            VARCHAR(200),
    photo_file          VARCHAR(500), -- storage path; image via files registry
    priority            maintenance_priority NOT NULL DEFAULT 'normal',
    status              maintenance_status NOT NULL DEFAULT 'new',
    assignee_person_id  UUID REFERENCES persons (id),
    recorded_cost       BIGINT CHECK (recorded_cost IS NULL OR recorded_cost >= 0),
    notes               TEXT,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
    closed_at           TIMESTAMPTZ
);

CREATE INDEX idx_maint_building ON maintenance_requests (building_id);
CREATE INDEX idx_maint_building_status ON maintenance_requests (building_id, status);
CREATE INDEX idx_maint_building_priority ON maintenance_requests (building_id, priority);
CREATE INDEX idx_maint_submitted_by ON maintenance_requests (submitted_by);
CREATE INDEX idx_maint_unit ON maintenance_requests (unit_id) WHERE unit_id IS NOT NULL;

COMMIT;
