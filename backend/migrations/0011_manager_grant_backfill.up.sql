-- 0011_manager_grant_backfill: repair zero-time manager grants.
--
-- Repository.GrantManager used FirstOrCreate(&UserBuilding{}) while
-- UserBuilding.GrantedAt lacked `autoCreateTime`, so GORM wrote Go's zero
-- time (0001-01-01) explicitly, defeating the column DEFAULT now(). Those
-- rows shipped over GET /buildings/{id}/managers as
-- "granted_at":"0001-01-01T00:00:00Z", which crashes the Flutter Jalali
-- conversion (DateException: out of computable range) and whitescreens the
-- building-managers route. The model tag is fixed alongside; this migration
-- heals rows already stored.
--
-- Backfill source: the building's own created_at — the creator auto-grant
-- (the only writer of zero times) is contemporaneous with building creation.

BEGIN;

UPDATE user_buildings ub
   SET granted_at = b.created_at
  FROM buildings b
 WHERE ub.building_id = b.id
   AND ub.granted_at < TIMESTAMPTZ '2000-01-01 00:00:00+00';

COMMIT;
