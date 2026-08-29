-- 0005_payments: US5 — payments (manual + gateway) and unit balances
-- (one row per unit, BR-01; reconciles to the spec §9 formula).
-- Per specs/001-building-management-mvp/data-model.md.

BEGIN;

CREATE TYPE payment_method AS ENUM ('manual', 'gateway');

-- payment_status: manual payments are manager-confirmed and land directly in
-- `verified`; gateway payments start `recorded` (pending verify) and become
-- `verified` (applied) or `failed` at callback verification. Reversal is a
-- status change — rows are never hard-deleted (BR-02).
CREATE TYPE payment_status AS ENUM ('recorded', 'verified', 'failed', 'reversed');

CREATE TABLE payments (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    building_id     UUID NOT NULL REFERENCES buildings (id),
    unit_id         UUID NOT NULL REFERENCES units (id),
    invoice_id      UUID REFERENCES invoices (id),
    method          payment_method NOT NULL,
    amount          BIGINT NOT NULL CHECK (amount > 0),
    paid_at         DATE NOT NULL,
    tracking_number VARCHAR(50),
    gateway         VARCHAR(20),
    authority       VARCHAR(100),
    recorded_by     UUID,
    status          payment_status NOT NULL DEFAULT 'recorded',
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_payments_unit ON payments (unit_id);
CREATE INDEX idx_payments_building ON payments (building_id);
CREATE INDEX idx_payments_invoice ON payments (invoice_id);
-- Callback lookup: the in-flight gateway payment by its provider authority.
CREATE UNIQUE INDEX uq_payments_authority ON payments (authority)
    WHERE status = 'recorded' AND authority IS NOT NULL;

-- unit_balances: the unit's current financial position (spec §9), recomputed
-- transactionally from invoices/adjustments/payments on every ledger event so
-- the components always reconcile:
--   balance = prior_debt + current_invoice_amount + late_fee_total
--             − paid_total − credit
-- credit_asset is derived money: verified payments beyond what invoices
-- absorbed (surplus overpayment), minus credit already embedded in issued
-- invoices — an asset outside the debt formula that the next calculation
-- snapshots as the invoice's credit_amount.
CREATE TABLE unit_balances (
    unit_id                UUID PRIMARY KEY REFERENCES units (id),
    prior_debt             BIGINT NOT NULL DEFAULT 0,
    current_invoice_amount BIGINT NOT NULL DEFAULT 0,
    late_fee_total         BIGINT NOT NULL DEFAULT 0,
    credit                 BIGINT NOT NULL DEFAULT 0,
    paid_total             BIGINT NOT NULL DEFAULT 0,
    credit_asset           BIGINT NOT NULL DEFAULT 0,
    balance                BIGINT NOT NULL DEFAULT 0,
    updated_at             TIMESTAMPTZ NOT NULL DEFAULT now()
);

COMMIT;
