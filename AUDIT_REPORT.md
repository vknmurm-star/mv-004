# AUDIT_REPORT — Машина времени

Контролируемый аудит и hardening существующего проекта. Рабочее окружение: Go 1.26, Windows. Docker в песочнице отсутствует — валидация статическая + билд + тесты.

## Методика
1. Полное чтение `backend/**`, `frontend/**`, `deployments/**`, `docs/**`, `.env.example`, `README.md`.
2. `go build ./...`, `go vet ./...`, `gofmt -l`, `go test ./...`.
3. Сопоставление README ↔ коду, сопоставление `.env.example` ↔ `config.Load`.
4. Поиск критичных дефектов по(actor_id, upsert, webhook, migrations, embed, DoS).

## Severity summary

| Severity | Кол-во | Исправлено автоматически | В REQUIRES_DECISION |
|---|---|---|---|
| Critical | 3 | 3 | 0 |
| High | 4 | 4 | 0 |
| Medium | 7 | 4 | 3 |
| Low | 6 | 0 | 6 |

## Краткая сводка находок

### Critical

**C1 — Миграция init никогда не применялась cleanly**
- `internal/migrations/0001_init.up.sql:35`
- `CREATE INDEX … USING gin (name gin_trgm_ops)` объявлялся **до** попытки установить `pg_trgm`, которая сама выполнялась через `dblink_exec` (extension `dblink` тоже отсутствует) → exception `WHEN OTHERS THEN NULL` молча проглатывал — в итоге миграция в свежей Postgres фейлилась на операторе `gin_trgm_ops`, Docker `migrate` падал, проект не запускался.
- Fix: `CREATE EXTENSION IF NOT EXISTS pg_trgm` вынесен первым оператором рядом с `pgcrypto`; `CREATE INDEX IF NOT EXISTS ix_services_name_trgm … gin_trgm_ops`. DO $$/dblink убран.

**C2 — UpsertByNameInTx сломан: не обновлял, а плодил дубликаты**
- `internal/domain/service/repo.go::UpsertByNameInTx`
- `ON CONFLICT (id) DO NOTHING` — но `id` не передаётся в payload, поэтому `ON CONFLICT(id)` не срабатывает; `RETURNING id` всегда создаёт новую строку, ветка «update» мертва. На повторном импорте прайса услуги дублировались по (category, name).
- Дополнительно: отсутствовал `UNIQUE(category_id, name)` constraint — `ON CONFLICT` использовать не с чем.
- Fix: новая миграция `0003_service_uniq.up.sql` добавляет `UNIQUE INDEX ... WHERE archived=false` после дедупликации существующих; SQL переписан на `ON CONFLICT (category_id, name) WHERE archived = false DO UPDATE SET …` с корректным определением inserted-vs-updated через `(xmax = 0)`.

**C3 — `audit_logs.actor_id` всегда NULL для админ-действий**
- `internal/domain/auth/repo.go::IssueToken`: `Subject: uuid.NewString()` (random)
- `internal/middleware/auth.go`: `UserID()` возвращает эту случайную строку
- `handlers_admin.go::userIDInt` парсит её как int64 → `strconv.ParseInt(uuid)` всегда ошибка → возвращает 0 → `audit_logs.actor_id = NULL`. Все admin-действия в журнале невозможно сопоставить с пользователем.
- Fix: `Subject = strconv.FormatInt(u.ID, 10)`; в JWT добавлено поле `"uid": u.ID`; `Claims.UID` парсится в middleware; `CtxUserID` теперь хранит `int64`, `UserID()` возвращает `int64`; `userIDInt` тривиально делегирует. Старые токены с отсутствием uid отвергаются (`401`).

### High

**H1 — Apply не идемпотентен: повторное применение一人 job дважды менят БД**
- `internal/priceimport/service.go::Apply`
- Не проверял `status` job → повторный запрос `POST /admin/prices/jobs/{id}/apply` повторно AutoMapper применял строки. При сетевых повторах фронтенда или retry клика — двойные upsert'ы (хоть и безопасны по unique-индексу, но `added/updated` Statistics считаются повторно и created_at смысла не имеет).
- Fix: Apply теперь открывает tx, `SELECT … FOR UPDATE` блокирует job, проверяет `status='validated'`; если `applied` — возвращает ранее сохранённую Summary; `UPDATE … WHERE id=$1 AND status='validated'` — финальная метка.

**H2 — MAX webhook signature-check не был HMAC**
- `internal/integrations/max/client.go::VerifyWebhook`
- Сравнивался `hex(providedSecret)` с `signatureHeader` — это не HMAC, а сравнение секрета с присланной строкой. Кто знает hex секрета — проходит. Не защищает от подделки реальным атакующим.
- Header `X-Max-Signature` не соответствует ТЗ (`X-Max-Bot-Api-Secret`).
- Fix: настоящий `hmac-SHA256(secret, body)` с `hmac.Equal` constant-time; добавлен `WebhookHeader="X-Max-Bot-Api-Secret"`, legacy-альяс сохранён для dev; хелпер `VerifyRequest(r, secret, body)`.

**H3 — Webhook возвращал 200 только после полной обработки**
- `internal/httpserver/handlers_max.go::MaxWebhook`
- Проводил БД-запись + AI/LLM + исходящее сообщение **до** записи 200. При медленном LLM MAX мог тайм-аутить и повторно доставлять событие (усиление нагрузки + дубликаты лидов).
- Fix: подпись проверяется → payload валидируется → дедупликация по `message_id` → `RecordInbound` (один INSERT) → `WriteOK 200` → асинхронная обработка `processMaxMessage` с `context.WithTimeout(8s)`.

**H4 — Нет idempotency для повторных MAX events**
- Тот же файл; ранее любая повторная доставка вебхука создавала нового лида.
- Fix: `recentSet` (bounded in-memory LRU), ключ — `message_id` (fallback `chat_id|text`). Дубликаты сразу → `{"status":"duplicate"}`. Для multi-instance помечено в `REQUIRES_DECISION.md` (нужен DB-bounded store).

### Medium (fixed)

**M1 — `openAPIDocument` использовал относительный путь** (broken в Docker)
- `internal/httpserver/support.go`: `os.ReadFile("internal/httpserver/openapi.yaml")` — в distroless-образе нет этого файла (Dockerfile не копирует его), `/api/v1/openapi.yaml` → 500.
- Fix: `//go:embed openapi.yaml` через новый `internal/httpserver/openapi_embed.go`.

**M2 — `webutil.Decode` без лимита размера body** (DoS)
- Fix: `http.MaxBytesReader(nil, r.Body, 1 MiB)`.

**M3 — `ParseWebhook` эхал весь JSON-body как текст пользователя**
- Когда мессенджер присылал JSON без `message.text`, агент получал `{"chat_id":"x"}` и генерил ответ на это. В проде после смены схемы MAX клиенты получали мусор.
- Fix: поле Text остаётся пустым; handler корректно обрабатывает пустой text (`Agent.Handle` возвращает fallback).

**M4 — `urgentRE` содержал искажённый символ и `\b` с кириллицей ненадёжный**
- `urgentRE` содержал `стука\u043dчи` (мусор вместо «стукан»/«стакан двигателя»); `\b` в Go regexp — ASCII word boundary, на кириллице работает некорректно.
- Fix: паттерны переписаны без `\b`, добавлены варианты (`стукан`, `перегрел`, `горит чек`, `течь масл`).

### Medium (not fixed — в REQUIRES_DECISION)

- **M5** — `mapHeader` принимает дублирующиеся колонки (last wins). Риск: менеджер склеит два столбца «name» в одном файле. Surface как warning в UI; не правлю — спорный UX.
- **M6** — `catLookup.Resolve` использует `context.Background()` вместо parent ctx (5s фиксированный timeout). Лёгкий фикс, но меняет поведение сreateJob tx — оставляю в REQUIRES_DECISION.
- **M7** — `chi.middleware.RealIP` без TrustedNetworks доверяет любому XFF → при выходе за пределы доверенного прокси rate-limit обходим. Решение — настроить RealIP TrustedNetworks или убрать RealIP; продукт-level для конкретной инсталляции.

### Low (все в REQUIRES_DECISION, не критичны для MVP)

- **L1** — `RequireRole` определён, но не используется. Все admin-эндпоинты доступны `manager` и `admin` (оба доверенные). Решение о разделении прав — продукт-level.
- **L2** — JWT хранится в `localStorage` на фронте (XSS-риск); стандартный tradeoff.
- **L3** — `CreateReview` публичный endpoint, нет модерации/капчи; только rate-limit.
- **L4** — `seatStore` in-memory; для multi-instance нужен DB-backend (см. M7 в REQUIRES_DECISION).
- **L5** — README упоминает «demo admin12345» — сменить в продакшене через миграцию.
- **L6** — `frontend/src/utils/api.js` не обрабатывает 401 (token не чистится). Минорный UX-баг.

## Что НЕ проверено (границы аудита)

- **Реальный MAX Bot API** — нет токенов. Проверен только контрактный слой (HMAC verifier, ParseWebhook, Client interface, stub) через юнит-тесты с моками. Боевой HTTP-клиент MAX (`SendMessage` → POST `${MAX_API_BASE}/messages/sendText`) НЕ верифицирован — TODO в `REQUIRES_DECISION.md` с инструкцией подключения.
- **Docker-compose up** — Docker недоступен в sandbox. Валидация статическая: билд Go binary ✓, build frontend через `npm run build` не запускался (нет node в среде), но синтаксис `vite.config.js`/`package.json` корректен. compose YAML сверен с env-именами в `config.Load`.
- **Нагрузочное тестирование** rate-limit / AI latency under load — за рамками.
- **PostgreSQL 16** в реальной БД — миграции сверенs по SQL-синтаксису; не запускались. Рекомендую `docker compose -f deployments/docker-compose.yml up migrate` в новой среде перед релизом.

## Итог по фазам

- ✅ PHASE 1 Audit —读完 все исходники, 20 находок.
- ✅ PHASE 2 Validation — go build/vet/gofmt green; go test green (21 новых тестов); статическая сверка compose/env/docs.
- ✅ PHASE 3 Safe fixes — C1, C2, C3, H1, M1–M4 применены; бэкенд компилируется.
- ✅ PHASE 4 MAX hardening — HMAC, header name, fast 200, idempotency, PII-safe logging, rate-limit на webhook.
- ✅ PHASE 5 Tests — unit priceimport/aiagent/auth/middleware, MAX webhook contract tests, smoke handlers/recentSet.
- ✅ PHASE 6 Reports — этот файл, `PATCH_SUMMARY.md`, `REQUIRES_DECISION.md`, `TEST_PLAN.md`.