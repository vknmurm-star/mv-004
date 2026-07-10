# PATCH_SUMMARY — Машина времени (hardening audit)

Все правки — safe critical/high/medium баги, без крупных архитектурных
переписываний. Спорные случаи вынесены в `REQUIRES_DECISION.md`. После правок:
`go build`, `go vet`, `go test ./...` — green.

## Изменённые файлы

### Backend — фикс критики
- `backend/internal/migrations/0001_init.up.sql` — вынесен `CREATE EXTENSION IF NOT EXISTS pg_trgm` upfront, удалён сломанный `DO $$ dblink_exec`-блок, `CREATE INDEX IF NOT EXISTS ix_services_name_trgm`. **(C1)**
- `backend/internal/migrations/0003_service_uniq.up.sql` — новый файл: `UNIQUE INDEX ux_services_category_name WHERE archived=false` после дедупликации существующих. **(C2)**
- `backend/internal/migrations/0003_service_uniq.down.sql` — домоход.
- `backend/internal/domain/service/repo.go` — `UpsertByNameInTx` переписан на `ON CONFLICT (category_id, name) WHERE archived = false DO UPDATE`, action определяется через `(xmax = 0)`. **(C2)**
- `backend/internal/domain/auth/repo.go` — `Subject = strconv.FormatInt(u.ID, 10)`, добавлено поле `uid` в JWT claims, удалён unused `uuid` импорт. **(C3)**
- `backend/internal/middleware/auth.go` — `Claims.UID` парсится из JWT; `CtxUserID` хранит `int64`; `RequireAuth` валидирует присутствие uid (reject legacy tokens без uid); `UserID()` возвращает `int64`. **(C3)**
- `backend/internal/httpserver/handlers_admin.go` — `userIDInt` упрощён до `middleware.UserID(r)` (раньше парсил uuid-строку в int64 → всегда 0). **(C3)**
- `backend/internal/priceimport/service.go` — `Apply` теперь в одной транзакции с `SELECT … FOR UPDATE` блокировкой job, проверяет `status='validated'`, возвращает хранязщуюся Summary для уже applied, `UPDATE … WHERE status='validated'`. **(H1)**

### Backend — MAX hardening
- `backend/internal/integrations/max/client.go` — `VerifyWebhook` переписан на настоящий HMAC-SHA256 с `hmac.Equal` (constant-time); добавлены константы `WebhookHeader="X-Max-Bot-Api-Secret"` и `WebhookHeaderLegacy`; добавлен `VerifyRequest(r, secret, body)`. **(H2)**
- `backend/internal/integrations/max/webhook.go` — `ParseWebhook` парсит `message_id` (string и numeric), убран опасный fallback "JSON as text" (M3), удалён неиспользуемый `slog` импорт.
- `backend/internal/httpserver/handlers_max.go` — **полная замена webhook handler**: verifyHMAC → parse → dedup(`recentSet` keyed по `message_id`) → `RecordInbound` (один INSERT) → `WriteOK 200` → **асинхронная** `processMaxMessage` с `context.WithTimeout(8s)`. **(H3, H4)**
- `backend/internal/httpserver/handlers.go` — в `handlers` добавлено поле `recentInbound *recentSet`.
- `backend/internal/httpserver/router.go` — webhook-роут получил rate-limit `10/min, burst 20` per-IP; конструктор `handlers` инъектирует `recentInbound`.
- `backend/internal/integrations/max/notifier.go` — PII-safe логирование: `NotifyAppointment`/`NotifyLead` логируют только `appointment_id`/`lead_id` и err, никогда не текст сообщения с телефоном клиента. **(M-PII)**

### Backend — прочие medium fixes
- `backend/internal/httpserver/openapi_embed.go` — новый файл: `//go:embed openapi.yaml` делает `/api/v1/openapi.yaml` рабочим в Docker. **(M1)**
- `backend/internal/httpserver/support.go` — удалён `os.ReadFile("internal/httpserver/openapi.yaml")` (заменён embed).
- `backend/pkg/webutil/webutil.go` — `Decode` оборачивает `Body` в `http.MaxBytesReader(nil, r.Body, 1<<20)` (1 MiB cap). **(M2)**
- `backend/internal/aiagent/agent.go` — паттерны `urgentRE`/`symptomRE`/`handoffRE`/`priceRE` переписаны без ASCII-`\b` (ненадёжно с кириллицей), исправлен мусорный символ в `стука\u043dчи`, добавлены варианты «стукан/перегрел/горит чек/течь масл». **(M4)**

### Backend — config
- `backend/internal/config/config.go` — prod validation усилена: `JWT_SECRET` ≥ 32 символа; если `MAX_ENABLED=true` в prod — обязательны `MAX_WEBHOOK_SECRET`, `MAX_API_TOKEN`, `MAX_NOTIFY_CHAT_ID`; если `AI_PROVIDER=openai` и включено в prod — обязателен `AI_API_KEY`. Добавлены `SetLogger`/`Log() *slog.Logger` для async-обработки логов.

### Backend — инфра/разное
- `backend/internal/middleware/ratelimit.go` — `clientIP` упрощён: использует `r.RemoteAddr` (который уже выставлен chi `RealIP`), а не повторно читает `X-Forwarded-For` (защита от IP-спуфинга для обхода rate-limit). **(P8 — частичный)**
- `backend/internal/middleware/ratelimit.go` — удалён неиспользуемый `net.IPv4len` мусорный блок.
- `backend/cmd/server/main.go` — `cfg.SetLogger(log)` связывает structured logger для async-логов.
- `backend/go.mod` — `go 1.24` (требуется `testing.T.Context()` в тестах; toolchain — go 1.26).

### Тесты (новые файлы)
- `backend/internal/priceimport/csv_test.go` — 8 тестов: `normalizePrice` edge-cases, `mapHeader` (missing/duplicate columns), `ParseCSV` (required columns, error rows, inactive(skip) rows, template/header invariant), `DetectFormat`.
- `backend/internal/integrations/max/client_test.go` — 12 тестов: HMAC verify (valid/bad/empty/constant-time), `VerifyRequest` по обоим header'ам, `ParseWebhook` (нормализация полей, nested `event`, отсутствие эха JSON, numeric `message_id`), stub client dry-run, deeplink.
- `backend/internal/integrations/max/helpers_test.go` — `testLogger()`.
- `backend/internal/aiagent/agent_test.go` — 6 тестов: rule-engine FSM полный flow, symptom skip, urgent escalation, handoff keyword, price-question never-invents-numbers, KB snippet fallback + disclaimers.
- `backend/internal/domain/auth/repo_test.go` — `mergeClaims` assert содержимого (uid/email/role/sub/iss).
- `backend/internal/middleware/auth_test.go` — 4 теста: `CtxUserID` хранит numeric uid после `RequireAuth`, rejects wrong secret, rejects missing token, `UserID()` 0 при отсутствии.
- `backend/internal/httpserver/webhook_smoke_test.go` — 5 тестов: `recentSet` dedup, bounded eviction, thread-safety, `handlers` construct, `readAllWithLimit` caps size.

## Запуск проверок

```bash
cd backend
go build ./...        # green
go vet ./...          # green
go test ./...         # ✓ 5 пакетов с тестами
gofmt -l internal cmd pkg  # пусто
```

## Что НЕ тронуто намеренно

- Фронтенд UI/UX (React-страницы) — без багов найдено, пометил L2/L6 (localStorage, 401-handling) в REQUIRES_DECISION.
- Архитектура `seatStore` (in-memory) — для multi-instance помечено в REQUIRES_DECISION.
- `RequireRole` (RBAC админ/менеджер) — не задействовано в роутере; продукт-уровневое решение в REQUIRES_DECISION.
- Реальный HTTP-клиент MAX (`SendMessage` → POST) — оставлен как extension point (инструкции в `docs/max-scenarios.md` и `REQUIRES_DECISION.md`).

## Что добавлено/усилено в плане безопасности

- HMAC webhook (раньше не HMAC)
- Prod env: обязательные секреты при MAX_ENABLED/AI
- Body size cap на JSON-декодер (DoS hardening)
- Webhook rate-limit
- Idempotency по message_id для повторных вебхуков
- Fast 200 OK на webhook
- HMAC constant-time compare
- PII не попадает в логи
- JWT содержит реальный user id → корректный audit trail