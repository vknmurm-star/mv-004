-- 0002_seed.down.sql
DELETE FROM knowledge_items;
DELETE FROM reviews;
DELETE FROM faq_items;
DELETE FROM services;
DELETE FROM service_categories;
DELETE FROM users WHERE email = 'admin@timemachine.ru';