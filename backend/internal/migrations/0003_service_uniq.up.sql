-- 0003_service_uniq.up.sql
-- Make service identity (category + name) unique so price import upserts
-- update existing rows instead of silently creating duplicates.
-- Existing duplicates (if any) are collapsed by keeping the latest id per
-- (category_id, name) and archiving the rest.

WITH dup AS (
    SELECT id, row_number() OVER (
        PARTITION BY category_id, name ORDER BY updated_at DESC, id DESC
    ) AS rn
    FROM services
)
UPDATE services SET archived = true, is_active = false
WHERE id IN (SELECT id FROM dup WHERE rn > 1);

CREATE UNIQUE INDEX IF NOT EXISTS ux_services_category_name
    ON services (category_id, name)
    WHERE archived = false;