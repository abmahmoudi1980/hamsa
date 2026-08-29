-- 0004_billing: US4 — billing periods, cost items, cost_item_shares,
-- invoices (+ immutability trigger guard), invoice_adjustments, invoice_items.
-- Per specs/001-building-management-mvp/data-model.md.

BEGIN;

CREATE TYPE billing_period_status AS ENUM ('draft', 'calculated', 'issued', 'closed');
CREATE TYPE cost_method AS ENUM ('equal', 'per_occupant', 'per_area', 'fixed', 'specific_units', 'combined');
CREATE TYPE late_fee_type AS ENUM ('none', 'fixed', 'percent', 'per_day');
CREATE TYPE invoice_status AS ENUM ('unpaid', 'partial', 'paid', 'expired', 'cancelled');
CREATE TYPE invoice_item_kind AS ENUM ('charge', 'late_fee', 'adjustment');
CREATE TYPE adjustment_kind AS ENUM ('debit', 'credit');

-- billing_periods: the charge cycle and its lifecycle state machine
-- (draft → calculated → issued → closed; calculated → draft allowed; issued →
-- draft forbidden).
CREATE TABLE billing_periods (
    id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    building_id    UUID NOT NULL REFERENCES buildings (id),
    title          VARCHAR(120) NOT NULL,
    start_date     DATE NOT NULL,
    end_date       DATE NOT NULL,
    due_date       DATE NOT NULL,
    late_fee_type  late_fee_type NOT NULL DEFAULT 'none',
    late_fee_value NUMERIC(12,2) NOT NULL DEFAULT 0,
    status         billing_period_status NOT NULL DEFAULT 'draft',
    calculated_at  TIMESTAMPTZ,
    issued_at      TIMESTAMPTZ,
    closed_at      TIMESTAMPTZ,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
    CHECK (end_date >= start_date AND due_date >= end_date)
);

CREATE INDEX idx_billing_periods_building ON billing_periods (building_id);

-- cost_items: one cost line of a period (spec §7). total_amount is Toman;
-- for `fixed` the service derives it as fixed_amount_per_unit × participants.
CREATE TABLE cost_items (
    id                    UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    period_id             UUID NOT NULL REFERENCES billing_periods (id),
    title                 VARCHAR(150) NOT NULL,
    total_amount          BIGINT NOT NULL CHECK (total_amount > 0),
    method                cost_method NOT NULL,
    fixed_amount_per_unit BIGINT,
    combo_weights         JSONB,
    include_vacant        BOOLEAN NOT NULL DEFAULT FALSE,
    unit_ids              UUID[] NOT NULL DEFAULT '{}',
    created_at            TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at            TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_cost_items_period ON cost_items (period_id);

-- cost_item_shares: calculation output per cost item per unit — the
-- reviewable breakdown (BR-08). exact_share keeps the rational result;
-- Σ(rounded_share) ≡ total_amount per cost item (BR-09).
CREATE TABLE cost_item_shares (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    cost_item_id    UUID NOT NULL REFERENCES cost_items (id) ON DELETE CASCADE,
    unit_id         UUID NOT NULL REFERENCES units (id),
    exact_share     NUMERIC(20,4) NOT NULL,
    rounded_share   BIGINT NOT NULL,
    inputs_snapshot JSONB NOT NULL DEFAULT '{}',
    UNIQUE (cost_item_id, unit_id)
);

CREATE INDEX idx_cost_item_shares_unit ON cost_item_shares (unit_id);

-- invoices: the immutable per-unit bill (spec §11). Amount columns are set at
-- calculation and never updated (BR-03/BR-05 — trigger guard below); after
-- issuance only status/paid_amount may change.
CREATE TABLE invoices (
    id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    invoice_number   VARCHAR(30) NOT NULL,
    building_id      UUID NOT NULL REFERENCES buildings (id),
    period_id        UUID NOT NULL REFERENCES billing_periods (id),
    unit_id          UUID NOT NULL REFERENCES units (id),
    base_amount      BIGINT NOT NULL DEFAULT 0,
    prior_debt       BIGINT NOT NULL DEFAULT 0,
    late_fee_amount  BIGINT NOT NULL DEFAULT 0,
    credit_amount    BIGINT NOT NULL DEFAULT 0,
    final_amount     BIGINT NOT NULL DEFAULT 0,
    issue_date       DATE,
    due_date         DATE,
    status           invoice_status NOT NULL DEFAULT 'unpaid',
    paid_amount      BIGINT NOT NULL DEFAULT 0 CHECK (paid_amount >= 0),
    inputs_frozen_at TIMESTAMPTZ,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Sequential per building (data-model.md); unique within the building.
CREATE UNIQUE INDEX uq_invoices_building_number ON invoices (building_id, invoice_number);
CREATE INDEX idx_invoices_period ON invoices (period_id);
CREATE INDEX idx_invoices_unit ON invoices (unit_id);
CREATE INDEX idx_invoices_building_status ON invoices (building_id, status);

-- invoice_adjustments: explicit corrections (FR-017/BR-03) — never mutate the
-- original amounts. Debit increases what the unit owes; credit decreases.
CREATE TABLE invoice_adjustments (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    invoice_id UUID NOT NULL REFERENCES invoices (id) ON DELETE CASCADE,
    kind       adjustment_kind NOT NULL,
    amount     BIGINT NOT NULL CHECK (amount > 0),
    reason     TEXT NOT NULL,
    created_by UUID,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_invoice_adjustments_invoice ON invoice_adjustments (invoice_id);

-- invoice_items: append-only displayable rows (charge lines, late fee,
-- adjustments). Existing rows are never updated or deleted; new adjustment
-- rows may be appended after issuance.
CREATE TABLE invoice_items (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    invoice_id    UUID NOT NULL REFERENCES invoices (id) ON DELETE CASCADE,
    kind          invoice_item_kind NOT NULL,
    title         VARCHAR(150) NOT NULL,
    cost_item_id  UUID REFERENCES cost_items (id),
    amount        BIGINT NOT NULL,
    method        cost_method,
    adjustment_id UUID REFERENCES invoice_adjustments (id),
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_invoice_items_invoice ON invoice_items (invoice_id);

-- ---------------------------------------------------------------------------
-- Immutability guard (BR-03/BR-05): amount/identity columns of an invoice are
-- INSERT-only. status/paid_amount may evolve (payment status machine), and
-- inputs_frozen_at is set at issuance. Deletion is blocked once the owning
-- period has been issued (cancelled rows are preserved — BR-10).
CREATE OR REPLACE FUNCTION billing_invoice_guard_update() RETURNS trigger AS $$
BEGIN
    -- Number assignment at issuance is the only permitted identity change:
    -- a DRAFT placeholder may become a real invoice number, nothing else.
    IF OLD.invoice_number LIKE 'DRAFT-%'
       AND NEW.invoice_number IS DISTINCT FROM OLD.invoice_number
       AND OLD.base_amount     IS NOT DISTINCT FROM NEW.base_amount
       AND OLD.prior_debt      IS NOT DISTINCT FROM NEW.prior_debt
       AND OLD.late_fee_amount IS NOT DISTINCT FROM NEW.late_fee_amount
       AND OLD.credit_amount   IS NOT DISTINCT FROM NEW.credit_amount
       AND OLD.final_amount    IS NOT DISTINCT FROM NEW.final_amount
       AND OLD.building_id     IS NOT DISTINCT FROM NEW.building_id
       AND OLD.period_id       IS NOT DISTINCT FROM NEW.period_id
       AND OLD.unit_id         IS NOT DISTINCT FROM NEW.unit_id
       AND OLD.due_date        IS NOT DISTINCT FROM NEW.due_date THEN
        RETURN NEW;
    END IF;
    IF OLD.base_amount     IS DISTINCT FROM NEW.base_amount
    OR OLD.prior_debt      IS DISTINCT FROM NEW.prior_debt
    OR OLD.late_fee_amount IS DISTINCT FROM NEW.late_fee_amount
    OR OLD.credit_amount   IS DISTINCT FROM NEW.credit_amount
    OR OLD.final_amount    IS DISTINCT FROM NEW.final_amount
    OR OLD.invoice_number  IS DISTINCT FROM NEW.invoice_number
    OR OLD.building_id     IS DISTINCT FROM NEW.building_id
    OR OLD.period_id       IS DISTINCT FROM NEW.period_id
    OR OLD.unit_id         IS DISTINCT FROM NEW.unit_id
    OR OLD.due_date        IS DISTINCT FROM NEW.due_date THEN
        RAISE EXCEPTION 'invoice amounts are immutable (BR-03/BR-05)';
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trg_invoices_immutable
    BEFORE UPDATE ON invoices
    FOR EACH ROW EXECUTE FUNCTION billing_invoice_guard_update();

CREATE OR REPLACE FUNCTION billing_invoice_guard_delete() RETURNS trigger AS $$
DECLARE
    period_status billing_period_status;
BEGIN
    SELECT status INTO period_status FROM billing_periods WHERE id = OLD.period_id;
    IF period_status IN ('issued', 'closed') THEN
        RAISE EXCEPTION 'issued invoices are preserved, never deleted (BR-10)';
    END IF;
    RETURN OLD;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trg_invoices_no_delete_after_issue
    BEFORE DELETE ON invoices
    FOR EACH ROW EXECUTE FUNCTION billing_invoice_guard_delete();

-- invoice_items: append-only. Existing rows are never updated or deleted; new
-- adjustment rows may be appended after issuance. Draft-period recalculation
-- cascades deletes (allowed only while the period is not issued).
CREATE OR REPLACE FUNCTION billing_invoice_item_guard_update() RETURNS trigger AS $$
BEGIN
    RAISE EXCEPTION 'invoice items are append-only (BR-03)';
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trg_invoice_items_no_update
    BEFORE UPDATE ON invoice_items
    FOR EACH ROW EXECUTE FUNCTION billing_invoice_item_guard_update();

CREATE OR REPLACE FUNCTION billing_invoice_item_guard_delete() RETURNS trigger AS $$
DECLARE
    period_status billing_period_status;
BEGIN
    SELECT status INTO period_status FROM billing_periods WHERE id = (
        SELECT period_id FROM invoices WHERE id = OLD.invoice_id
    );
    IF period_status IN ('issued', 'closed') THEN
        RAISE EXCEPTION 'issued invoice items are preserved (BR-03)';
    END IF;
    RETURN OLD;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trg_invoice_items_no_delete_after_issue
    BEFORE DELETE ON invoice_items
    FOR EACH ROW EXECUTE FUNCTION billing_invoice_item_guard_delete();

-- invoice_adjustments: append-only corrections; drafts cascade-delete with
-- their invoice, issued periods keep every row.
CREATE OR REPLACE FUNCTION billing_adjustment_guard_update() RETURNS trigger AS $$
BEGIN
    RAISE EXCEPTION 'adjustments are append-only (BR-03)';
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trg_invoice_adjustments_no_update
    BEFORE UPDATE ON invoice_adjustments
    FOR EACH ROW EXECUTE FUNCTION billing_adjustment_guard_update();

CREATE OR REPLACE FUNCTION billing_invoice_item_guard_insert() RETURNS trigger AS $$
DECLARE
    period_status billing_period_status;
BEGIN
    IF NEW.kind <> 'adjustment' THEN
        SELECT status INTO period_status FROM billing_periods WHERE id = (
            SELECT period_id FROM invoices WHERE id = NEW.invoice_id
        );
        IF period_status IN ('issued', 'closed') THEN
            RAISE EXCEPTION 'only adjustment items may be appended after issuance (BR-03)';
        END IF;
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trg_invoice_items_no_insert_after_issue
    BEFORE INSERT ON invoice_items
    FOR EACH ROW EXECUTE FUNCTION billing_invoice_item_guard_insert();

COMMIT;

