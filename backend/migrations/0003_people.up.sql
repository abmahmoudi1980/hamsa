-- 0003_people: US3 — persons, occupancies (end-dated history), and
-- occupant_count_history (independent as-of data points).
-- Per specs/001-building-management-mvp/data-model.md.

BEGIN;

CREATE TYPE occupancy_relationship AS ENUM ('owner', 'tenant', 'non_resident_owner');

-- persons: natural people linked to units (owner / tenant / non-resident
-- owner); scoped per building for P0 simplicity.
CREATE TABLE persons (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    building_id  UUID        NOT NULL REFERENCES buildings (id),
    full_name    VARCHAR(150) NOT NULL,
    phone        VARCHAR(11),
    national_id  VARCHAR(10),
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at   TIMESTAMPTZ
);

CREATE INDEX idx_persons_building ON persons (building_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_persons_phone ON persons (phone);
CREATE INDEX idx_persons_national_id ON persons (national_id);

-- occupancies: dated person↔unit link; records are CLOSED (end-dated),
-- never deleted when a resident changes (FR-007). end_date NULL = active.
CREATE TABLE occupancies (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    unit_id       UUID NOT NULL REFERENCES units (id),
    person_id     UUID NOT NULL REFERENCES persons (id),
    relationship  occupancy_relationship NOT NULL,
    start_date    DATE NOT NULL,
    end_date      DATE,
    is_active     BOOLEAN NOT NULL DEFAULT TRUE,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    CHECK (end_date IS NULL OR end_date >= start_date)
);

CREATE INDEX idx_occupancies_unit ON occupancies (unit_id);
CREATE INDEX idx_occupancies_person ON occupancies (person_id);
-- Only one active tenant per unit at a time (data-model.md rule). Other
-- relationships (owner / non_resident_owner) may coexist.
CREATE UNIQUE INDEX uq_occupancies_active_tenant
    ON occupancies (unit_id)
    WHERE relationship = 'tenant' AND end_date IS NULL;

CREATE INDEX idx_occupancies_unit_dates ON occupancies (unit_id, start_date);

-- occupant_count_history: independent occupant-count data points feeding the
-- charge engine (FR-008, BR-04); current value = max effective_from ≤
-- reference date.
CREATE TABLE occupant_count_history (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    unit_id         UUID NOT NULL REFERENCES units (id),
    occupant_count  INT  NOT NULL CHECK (occupant_count >= 0),
    effective_from  DATE NOT NULL,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    created_by      UUID
);

CREATE INDEX idx_occupant_count_unit_from
    ON occupant_count_history (unit_id, effective_from DESC);

COMMIT;
