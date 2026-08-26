-- 0002_buildings: US2 — buildings, units, user_buildings (manager scope).
-- Per specs/001-building-management-mvp/data-model.md.

BEGIN;

CREATE TYPE unit_status AS ENUM ('active', 'vacant', 'occupied', 'inactive');

-- buildings: manager-registered building registry.
CREATE TABLE buildings (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name            VARCHAR(150) NOT NULL,
    address         TEXT,
    block_count     INT          NOT NULL DEFAULT 0 CHECK (block_count >= 0),
    floor_count     INT          NOT NULL DEFAULT 0 CHECK (floor_count >= 0),
    unit_count      INT          NOT NULL DEFAULT 0 CHECK (unit_count >= 0),
    built_year      INT,
    manager_phone   VARCHAR(11),
    emergency_phone VARCHAR(11),
    notes           TEXT,
    created_at      TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ  NOT NULL DEFAULT now(),
    deleted_at      TIMESTAMPTZ
);

-- units: unit registry; number unique per building among non-deleted rows
-- (FR-003). Soft delete preserves financial history (spec §21).
CREATE TABLE units (
    id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    building_id      UUID        NOT NULL REFERENCES buildings (id),
    number           VARCHAR(20) NOT NULL,
    block            VARCHAR(20),
    floor            INT         NOT NULL DEFAULT 0,
    area_m2          INT         NOT NULL CHECK (area_m2 > 0),
    parking_count    INT         NOT NULL DEFAULT 0 CHECK (parking_count >= 0),
    parking_numbers  TEXT,
    storage_count    INT         NOT NULL DEFAULT 0 CHECK (storage_count >= 0),
    storage_numbers  TEXT,
    status           unit_status NOT NULL DEFAULT 'active',
    notes            TEXT,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at       TIMESTAMPTZ
);

CREATE UNIQUE INDEX uq_units_building_number_active
    ON units (building_id, number)
    WHERE deleted_at IS NULL;

CREATE INDEX idx_units_building ON units (building_id);
CREATE INDEX idx_units_building_filters ON units (building_id, block, floor, status);

-- user_buildings: manager permission scope (FR-037).
CREATE TABLE user_buildings (
    user_id     UUID NOT NULL REFERENCES users (id),
    building_id UUID NOT NULL REFERENCES buildings (id),
    granted_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (user_id, building_id)
);

COMMIT;
