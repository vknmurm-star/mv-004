# Машина времени — автомастерская (сайт + mini-CRM)

Современный сайт-витрина с мини-CRM для автосервиса «Машина времени»:
публичный каталог услуг и прайса, онлайн-запись, админ-панель управления
услугами и ценами с импортом/экспортом CSV/XLSX, интеграция с мессенджером
MAX (бот + ИИ-агент) и API-first бэкенд на Go.

## Технологии
- **Backend:** Go 1.22, chi, pgx/v5 (pool), JWT, bcrypt
- **DB:** PostgreSQL 16 (миграции на чистом SQL, golang-migrate-совместимые)
- **Frontend:** React 18 + Vite (SPA, mobile-first, светлая/тёмная тема)
- **Импорт прайса:** CSV + XLSX (excelize), предпросмотр, подсветка ошибок
- **MAX:** интеграционный слой с проверкой подписи webhook, сухим запуском
  (rule-based детерминированный бот + опциональный LLM RAG по базе знаний)
- **Infra:** Docker, docker-compose
- **API:** REST JSON, OpenAPI/Swagger (`/api/v1/openapi.yaml`)

## Структура проекта
```
timemachine-auto/
├─ backend/
│  ├─ cmd/{server,migrate}/main.go
│  ├─ internal/
│  │  ├─ config/ apperror/ middleware/ httpserver/
│  │  ├─ domain/         (category/auth/booking/lead/faq/knowledge/review/audit)
│  │  ├─ priceimport/ aiagent/ integrations/max/ migrations/
│  ├─ pkg/ (db webutil)
│  ├─ go.mod / Dockerfile
├─ frontend/             (React 18 + Vite, все страницы + админка)
├─ deployments/docker-compose.yml
├─ docs/                 (db-schema.md, max-scenarios.md)
└─ .env.example
```

## Быстрый старт (Docker)
```bash
cp .env.example .env
docker compose -f deployments/docker-compose.yml up --build
```

Чтобы всё поднимать одним `docker compose up`, создайте в корне симлинк или
копию compose-файла; здесь compose лежит в `deployments/`.
- Frontend: http://localhost:3000
- API:      http://localhost:8080/api/v1/health
- Админка:  http://localhost:3000/admin/login
  (demo: `admin@timemachine.ru` / `admin12345`)

compose поднимает Postgres, прогоняет `up`-миграции, затем бэкенд и фронт.

## Локальная разработка (без Docker)
```bash
# БД
docker run -d --name tm-pg -e POSTGRES_PASSWORD=timemachine -e POSTGRES_USER=timemachine \
  -e POSTGRES_DB=timemachine -p 5432:5432 postgres:16-alpine

# Бэкенд
cd backend
go mod tidy
go run ./cmd/migrate -direction=up
go run ./cmd/server     # :8080

# Фронтенд (проксирует /api на :8080)
cd ../frontend
npm install
npm run dev            # :5173
```

## Управление прайсом (админка)
1. /admin/prices/import → drag&drop CSV/XLSX.
2. Предпросмотр: сколько добавлено/обновлено/пропущено/ошибок, строки с
   ошибками подсвечиваются.
3. «Подтвердить и применить» — атомарный import inside one transaction.
4. Шаблон: «⬇ Шаблон CSV»; экспорт текущих → «⬇ Экспорт».
5. Журнал импортов ниже на той же странице; детальные строки по job через
   `GET /api/v1/admin/prices/jobs/{id}`.
6. Ручное редактирование услуг — /admin/services.

CSV-формат (шаблон идёт в той же колонке):
```
category,name,description,price,currency,duration_minutes,is_from_price,is_active
Тормозная система,Замена колодок (оси),Передние или задние работа за ось,3500,RUB,45,да,1
```
Массовое обновление идемпотентно по (category + name). Цена хранится в копейках;
парсер нормализует «1 500 ₽», «от 1500», «1500.00».

## MAX-интеграция и AI-агент
Подробно в `docs/max-scenarios.md`. Кратко:
- `POST /api/v1/max/webhook` — входящий вебхук с проверкой подписи.
- `MAX_ENABLED=false` → сообщения только логируются (dry-run), удобно для
  разработки.
- Бот: rule-based + опциональный LLM-RAG по `knowledge_items`
  (`AI_ENABLED=true`, `AI_PROVIDER=openai|stub`).
- База знаний редактируется в /admin/knowledge.
- Дисклеймер о диагнозе/цене добавляется ко всем AI-ответам.

## API
OpenAPI документ: `GET /api/v1/openapi.yaml`. Основные эндпоинты:

Публичные: `GET /categories`, `/services`, `/services/{id}`, `/faq`,
`/reviews`; `POST /appointments`, `/leads`, `/reviews`; `GET /max/deeplink`,
`POST /max/webhook`.

Админ (Bearer JWT): `/admin/login`, `/admin/services` (CRUD),
`/admin/prices/{template,export,import,jobs,jobs/{id}/apply}`,
`/admin/appointments` (list/patch), `/admin/leads`, `/admin/faq`,
`/admin/knowledge`, `/admin/reviews`, `/admin/audit`.

## Безопасность
- Авторизация админа через JWT (HMAC, бинарник из `cmd/server`).
- Секреты — переменные окружения (см. `.env.example`); в production
  `JWT_SECRET` и `MAX_WEBHOOK_SECRET` обязательны.
- CORS allow-list из `CORS_ALLOWED_ORIGINS`.
- Rate limiting: публичные формы — на IP/час; API — на IP/мин.
- Secure headers + HSTS + CSP в production.
- Валидация всех DTO на уровне доменных репозиториев (`Validate()`).

## Расширяемость
- Каждый домен — отдельный пакет с `Repo` и `Validation`.
- Внешние интеграции (MAX, LLM) за интерфейсами; замена заглушек реальными
  HTTP-клиентами не требует правок хендлеров.
- `seatStore` (состояние чата) заменяется таблицей `sessions`.
- Добавление нового источника лидов = новый `lead.Channel` + хендлер.

## Demo-креды
- admin@timemachine.ru / admin12345 — сменить до продакшена!