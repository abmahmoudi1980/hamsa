-- 0005_payments down: drop US5 payment/balance objects.

BEGIN;

DROP TABLE IF EXISTS unit_balances;
DROP TABLE IF EXISTS payments;
DROP TYPE IF EXISTS payment_status;
DROP TYPE IF EXISTS payment_method;

COMMIT;
