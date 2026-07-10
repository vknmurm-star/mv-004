-- 0001_init.down.sql
DROP TRIGGER IF EXISTS appointments_touch ON appointments;
DROP TRIGGER IF EXISTS services_touch ON services;
DROP FUNCTION IF EXISTS touch_updated_at();

DROP TABLE IF EXISTS audit_logs;
DROP TABLE IF EXISTS sessions;
DROP TABLE IF EXISTS max_messages;
DROP TABLE IF EXISTS reviews;
DROP TABLE IF EXISTS knowledge_items;
DROP TABLE IF EXISTS faq_items;
DROP TABLE IF EXISTS leads;
DROP TABLE IF EXISTS appointments;
DROP TABLE IF EXISTS price_import_rows;
DROP TABLE IF EXISTS price_import_jobs;
DROP TABLE IF EXISTS users;
DROP TABLE IF EXISTS services;
DROP TABLE IF EXISTS service_categories;