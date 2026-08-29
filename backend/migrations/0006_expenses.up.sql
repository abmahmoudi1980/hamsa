-- 0006_expenses: US6 — categorized building expenses with receipts and an
-- approval workflow (FR-026) feeding the financial report (FR-027).
-- Per specs/001-building-management-mvp/data-model.md.

BEGIN;

CREATE TYPE expense_category AS ENUM (
    'water', 'electricity', 'gas', 'elevator', 'cleaning',
    'security', 'repair', 'insurance', 'equipment', 'other'
);

-- approval_status: an expense is recorded as pending and moves to
-- approved/rejected through the manager's update (approval workflow).
CREATE TYPE expense_approval AS ENUM ('pending', 'approved', 'rejected');

-- expenses: soft delete preserves the financial history (spec §21) —
-- deleted rows vanish from lists and reports but never from the ledger.
CREATE TABLE expenses (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    building_id     UUID NOT NULL REFERENCES buildings (id),
    title           VARCHAR(150) NOT NULL,
    category        expense_category NOT NULL,
    amount          BIGINT NOT NULL CHECK (amount > 0),
    expense_date    DATE NOT NULL,
    description     TEXT,
    payer_person_id UUID REFERENCES persons (id),
    receipt_file    VARCHAR(500), -- storage path; image/PDF (files registry)
    approval_status expense_approval NOT NULL DEFAULT 'pending',
    created_by      UUID,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at      TIMESTAMPTZ
);

CREATE INDEX idx_expenses_building ON expenses (building_id);
CREATE INDEX idx_expenses_building_date ON expenses (building_id, expense_date) WHERE deleted_at IS NULL;
CREATE INDEX idx_expenses_building_approval ON expenses (building_id, approval_status) WHERE deleted_at IS NULL;

COMMIT;
