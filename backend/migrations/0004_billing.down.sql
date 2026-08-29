-- 0004_billing down: drop billing schema objects.

BEGIN;

DROP TRIGGER IF EXISTS trg_invoice_adjustments_no_update ON invoice_adjustments;
DROP FUNCTION IF EXISTS billing_adjustment_guard_update();
DROP TRIGGER IF EXISTS trg_invoice_items_no_delete_after_issue ON invoice_items;
DROP TRIGGER IF EXISTS trg_invoice_items_no_update ON invoice_items;
DROP TRIGGER IF EXISTS trg_invoice_items_no_insert_after_issue ON invoice_items;
DROP FUNCTION IF EXISTS billing_invoice_item_guard_insert();
DROP FUNCTION IF EXISTS billing_invoice_item_guard_delete();
DROP FUNCTION IF EXISTS billing_invoice_item_guard_update();
DROP TRIGGER IF EXISTS trg_invoices_no_delete_after_issue ON invoices;
DROP TRIGGER IF EXISTS trg_invoices_immutable ON invoices;
DROP FUNCTION IF EXISTS billing_invoice_guard_delete();
DROP FUNCTION IF EXISTS billing_invoice_guard_update();

DROP TABLE IF EXISTS invoice_items CASCADE;
DROP TABLE IF EXISTS invoice_adjustments CASCADE;
DROP TABLE IF EXISTS invoices CASCADE;
DROP TABLE IF EXISTS cost_item_shares CASCADE;
DROP TABLE IF EXISTS cost_items CASCADE;
DROP TABLE IF EXISTS billing_periods CASCADE;

DROP TYPE IF EXISTS adjustment_kind;
DROP TYPE IF EXISTS invoice_item_kind;
DROP TYPE IF EXISTS invoice_status;
DROP TYPE IF EXISTS late_fee_type;
DROP TYPE IF EXISTS cost_method;
DROP TYPE IF EXISTS billing_period_status;

COMMIT;
