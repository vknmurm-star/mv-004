-- 0002_seed.up.sql
-- Demo content for «Машина времени» auto service.
-- Admin password: "admin12345" (bcrypt hash). Replace in production!

INSERT INTO users (email, password_hash, role, name, is_active) VALUES
('admin@timemachine.ru',
'$2a$10$2R8dQqLk8YX9v8Ye3Lx2.O3kUMeO0Vc5t9bBqYw1Cb3XnLf2h.o8m',
'admin', 'Администратор', true)
ON CONFLICT (email) DO NOTHING;

-- Categories
INSERT INTO service_categories (name, slug, sort_order) VALUES
('Диагностика',      'diagnostics',   10),
('Техническое обслуживание', 'maintenance', 20),
('Тормозная система', 'brakes',        30),
('Двигатель',         'engine',        40),
('Подвеска и рулевое','suspension',    50),
('Электрика',         'electric',      60),
('Шиномонтаж',        'tires',         70),
('Кузов и покраска',  'body',          80),
('Дополнительно',     'extra',         90)
ON CONFLICT (slug) DO NOTHING;

-- Services (price_cents = price * 100, integer cents)
INSERT INTO services (category_id, name, description, price_cents, currency, duration_minutes, is_from_price, is_active, sort_order) VALUES
((SELECT id FROM service_categories WHERE slug='diagnostics'), 'Компьютерная диагностика', 'Считывание и расшифровка ошибок ЭБУ всех систем', 150000, 'RUB', 60, false, true, 10),
((SELECT id FROM service_categories WHERE slug='diagnostics'), 'Диагностика подвески', 'Проверка ходовой части, оценка состояния узлов', 100000, 'RUB', 45, true, true, 20),
((SELECT id FROM service_categories WHERE slug='diagnostics'), 'Предпродажная диагностика', 'Полный осмотр перед покупкой авто — 150+ пунктов', 400000, 'RUB', 120, false, true, 30),

((SELECT id FROM service_categories WHERE slug='maintenance'), 'Замена масла ДВС', 'Работа + промывка, без расходников', 60000, 'RUB', 30, true, true, 10),
((SELECT id FROM service_categories WHERE slug='maintenance'), 'ТО-1 (малое)', 'Масло, фильтры масляный/воздушный/салонный, осмотр', 350000, 'RUB', 90, true, true, 20),
((SELECT id FROM service_categories WHERE slug='maintenance'), 'ТО-2 (большое)', 'Расширенное ТО по регламенту производителя', 900000, 'RUB', 180, true, true, 30),
((SELECT id FROM service_categories WHERE slug='maintenance'), 'Замена охлаждающей жидкости', 'С промывкой системы, под зарядкой', 400000, 'RUB', 60, false, true, 40),

((SELECT id FROM service_categories WHERE slug='brakes'), 'Замена колодок (оси)', 'Передние или задние, работа за ось', 350000, 'RUB', 45, true, true, 10),
((SELECT id FROM service_categories WHERE slug='brakes'), 'Замена тормозных дисков', 'За ось, с проверкой биения', 500000, 'RUB', 90, true, true, 20),
((SELECT id FROM service_categories WHERE slug='brakes'), 'Прокачка тормозов', 'Замена тормозной жидкости, удаление воздуха', 200000, 'RUB', 45, false, true, 30),

((SELECT id FROM service_categories WHERE slug='engine'), 'Замена ремня ГРМ', 'С роликами, по регламенту', 1200000, 'RUB', 360, true, true, 10),
((SELECT id FROM service_categories WHERE slug='engine'), 'Замена свечей зажигания', 'Работа за комплект', 400000, 'RUB', 60, true, true, 20),
((SELECT id FROM service_categories WHERE slug='engine'), 'Мойка двигателя', 'С просушкой и защитой электрики', 500000, 'RUB', 60, false, true, 30),

((SELECT id FROM service_categories WHERE slug='suspension'), 'Замена амортизаторов', 'За штуку, работа', 300000, 'RUB', 60, true, true, 10),
((SELECT id FROM service_categories WHERE slug='suspension'), 'Развал-схождение 3D', 'Регулировка углов на стенде', 250000, 'RUB', 45, false, true, 20),
((SELECT id FROM service_categories WHERE slug='suspension'), 'Замена рулевых наконечников', 'За штуку, работа', 200000, 'RUB', 45, true, true, 30),

((SELECT id FROM service_categories WHERE slug='electric'), 'Диагностика электрики', 'Поиск утечек тока, проверка цепей', 150000, 'RUB', 60, true, true, 10),
((SELECT id FROM service_categories WHERE slug='electric'), 'Замена аккумулятора', 'С проверкой генератора и сбросом ошибок', 200000, 'RUB', 30, false, true, 20),
((SELECT id FROM service_categories WHERE slug='electric'), 'Установка сигнализации', 'Базовый комплект, работа', 800000, 'RUB', 180, true, true, 30),

((SELECT id FROM service_categories WHERE slug='tires'), 'Шиномонтаж R13–R16', 'С балансировкой, за колесо', 200000, 'RUB', 15, false, true, 10),
((SELECT id FROM service_categories WHERE slug='tires'), 'Шиномонтаж R17–R20', 'С балансировкой, за колесо', 350000, 'RUB', 20, false, true, 20),
((SELECT id FROM service_categories WHERE slug='tires'), 'Хранение шин (сезон)', 'За комплект, в сезон', 600000, 'RUB', 0, true, true, 30),

((SELECT id FROM service_categories WHERE slug='body'), 'Локальная покраска', 'Один элемент, материал+работа', 600000, 'RUB', 240, true, true, 10),
((SELECT id FROM service_categories WHERE slug='body'), 'Полировка кузова', 'Восстановительная, кузов целиком', 900000, 'RUB', 240, false, true, 20),
((SELECT id FROM service_categories WHERE slug='body'), 'Удаление вмятин без покраски (PDR)', 'За одну вмятину', 500000, 'RUB', 60, true, true, 30),

((SELECT id FROM service_categories WHERE slug='extra'), 'Эвакуатор', 'По городу, до 30 км', 200000, 'RUB', 0, true, true, 10),
((SELECT id FROM service_categories WHERE slug='extra'), 'Предоставление подменного авто', 'На время ремонта, сутки', 150000, 'RUB', 0, true, true, 20)
ON CONFLICT DO NOTHING;

-- FAQ
INSERT INTO faq_items (question, answer, category, sort_order, is_published) VALUES
('Как записаться на ремонт?', 'Онлайн через форму «Запись», по телефону или через чат-бот в мессенджере MAX.', 'Запись', 10, true),
('Можно ли приехать без записи?', 'Можно, но время и наличие мастера не гарантируем. Лучше записаться заранее — это бесплатно.', 'Запись', 20, true),
('Дают ли гарантию на работы?', 'Да, на работы — 6 месяцев, на расходники — согласно гарантии производителя.', 'Гарантия', 10, true),
('Можно ли привезти свои запчасти?', 'Можно, но мы не несём гарантий на сторонние запчасти. Работа по нашим — с гарантией.', 'Запчасти', 10, true),
('Сколько стоит диагностика?', 'Компьютерная диагностика — 1500 ₽, диагностика подвески — от 1000 ₽. Точную смету дадим после осмотра.', 'Цены', 10, true),
('Какие способы оплаты?', 'Наличные, карта, перевод по QR (СБП), безнал для юр. лиц.', 'Оплата', 10, true),
('Выдаёте подменное авто?', 'Да, при ремонте длительностью более одного дня — по запросу и наличию.', 'Сервис', 10, true)
ON CONFLICT DO NOTHING;

-- Reviews demo
INSERT INTO reviews (author, rating, body, car_info, is_published) VALUES
('Андрей', 5, 'Прозошли проблему со стойками за один день. Понятная смета, без накруток.', 'Kia Ceed 2018', true),
('Марина', 4, 'Записалась через MAX, перезвонили быстро. Сделали ТО вовремя.', 'Toyota Corolla 2020', true),
('Игорь', 5, 'Нашли утечку тока, которую два других сервиса не заметили. Рекомендую.', 'Volkswagen Tiguan 2017', true)
ON CONFLICT DO NOTHING;

-- Knowledge base for AI agent
INSERT INTO knowledge_items (title, body, tags, source) VALUES
('Часы работы и адрес', 'Автомастерская «Машина времени»: Москва, ул. Гаражная 12. Пн–Сб 09:00–20:00, Вс — выходной. Телефон +7 (495) 000-00-00.', ARRAY['general','hours','address'], 'manual'),
('Запись на ремонт', 'Запись онлайн на сайте, по телефону или в MAX. Желательно указать марку, модель, госномер и кратко проблему. Подтверждение приходит в течение часа в рабочее время.', ARRAY['booking'], 'manual'),
('Гарантия', 'Гарантия на работы — 6 месяцев, на расходники — по гарантии производителя. Гарантия действует при соблюдении рекомендаций мастера.', ARRAY['warranty','guarantee'], 'manual'),
('Стоимость диагностики', 'Компьютерная диагностика — 1500 ₽, диагностика подвески — от 1000 ₽, предпродажная — 4000 ₽. Точную смету формируем после осмотра.', ARRAY['price','diagnostics'], 'manual'),
('Безопасность переписки с агентом', 'ИИ-агент не ставит диагноз по переписке, а помогает сориентироваться, подобрать услугу и записаться. Точный диагноз — после осмотра автомобиля мастером.', ARRAY['ai','safety','disclaimer'], 'manual'),
('Тормоза: когда ехать срочно', 'Мягкая педаль, длинный тормозной путь, скрежет, пульсация или утечь тормозной жидкости — записывайтесь как можно скорее. Это вопрос безопасности.', ARRAY['brakes','urgent','safety'], 'manual')
ON CONFLICT DO NOTHING;