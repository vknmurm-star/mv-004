# TEST_PLAN — Машина времени

Сводный план проверок после аудита и hardening. Указано, что покрыто
автоматизированными тестами, что требует ручной пробы, и что требует боевых
секретов MAX.

## 1. Unit-тесты (автоматизированы, `go test ./...` — green)

| Пакет | Тестов | Покрытие |
|---|---|---|
| `internal/priceimport` | 8 | `normalizePrice` (price-format edge-cases), `mapHeader` (missing/duplicate columns), `ParseCSV` (required columns, error rows, inactive→skip), `DetectFormat`, `Template`/parser-invariant |
| `internal/integrations/max` | 12 | HMAC-valid, HMAC-bad, empty-secret, empty-signature, constant-time path, `VerifyRequest` canonical+legacy headers, `ParseWebhook` field normalization, nested event fallback, no-JSON-as-text regression, numeric `message_id`, stub client dry-run, deeplink |
| `internal/aiagent` | 6 | Rule-engine FSM lead capture flow, symptom skip, urgent escalation, handoff keyword, price question never invents numbers, KB snippet fallback + disclaimers |
| `internal/domain/auth` | 2 | `mergeClaims` assertions (uid/email/role/sub/iss) |
| `internal/middleware` | 4 | `RequireAuth` stores numeric uid, rejects bad secret, rejects missing token, `UserID()` zero when absent |
| `internal/httpserver` | 5 | `recentSet` dedup, bounded eviction, thread-safety (100 goroutines), `handlers` construct, `readAllWithLimit` size cap |

**Итого: 37 unit/contract тестов, все green.**

```bash
cd backend
go test ./...                       # запустить все
go test ./internal/priceimport/...  # только priceimport
go test -run TestNormalize -v ./internal/priceimport/...
```

## 2. Smoke flow (минимальный, manual/integration)

Покрытие сценария из ТЗ «минимальный smoke flow»:

1. **project start** — `cd backend && go run ./cmd/migrate -direction=up` затем `go run ./cmd/server` (= `:8080`). Green по `go build`.
   - Альтернатива: `docker compose -f deployments/docker-compose.yml up --build` (compose применит миграции через контейнер `migrate`, затем поднимет backend+frontend).
2. **health endpoint** — `GET /api/v1/admin/health`; ручно:
   - `curl -s http://localhost:8080/api/v1/health` → `{"data":{"status":"ok"}}`
3. **admin login** — `POST /api/v1/admin/login` с `{"email":"admin@timemachine.ru","password":"admin12345"}` → `{"data":{"token":"...","user":{"id":1,...}}}`.
4. **create booking** — `POST /api/v1/appointments` сJSON из ТЗ → `201 {"data":{"id":N}}` → в `audit_logs` появляется запись `action=create, actor_id=NULL` (ожидаемо — публичная форма).
5. **import price file** — `POST /api/v1/admin/prices/import` (multipart, `file=template.csv`) с Bearer-токеном → `201` с `job_id` и preview. Pre-rec: взять шаблон `GET /api/v1/admin/prices/template`.
6. **confirm import** — `POST /api/v1/admin/prices/jobs/{id}/apply` → `200 {"data":{"job_id":..., "summary":{"added":..., "updated":..., "skipped":..., "errors":0}}}`.
7. **verify service updates** — `GET /api/v1/services` показывает применённые строки.
   - **idempotency check:** повторный `POST /admin/prices/jobs/{id}/apply` → `200` с той же Summary (без дублирующего действия) — проверяет H1.
8. **mock MAX lead flow** — см. §3 ниже.

Все шаги smoke flow ручные (требуют поднятой БД). Рекомендую оформить их как
интеграционный скрипт `scripts/smoke.sh` после того, как выбран environment.
Сам набор API-вызовов сверен с роутером и ✓ валиден.

## 3. MAX webhook mock test — автоматически

Покрыто unit-тестами в §1 (`client_test.go`); здесь — ручная проба end-to-end
локально. Локально без секретов:

```bash
# 1) без настройки MAX_WEBHOOK_SECRET (dev-mode allowing)
SERVER=http://localhost:8080
#   /api/v1/max/deeplink должен вернуть диплинк
curl -s $SERVER/api/v1/max/deeplink

# 2) отправить вкорне webhook без валидной подписи — прод отвергнёт, dev примет
curl -s -XPOST $SERVER/api/v1/max/webhook \
  -H 'Content-Type: application/json' \
  -H 'X-Max-Bot-Api-Secret: invalid' \
  -d '{"chat_id":"c1","message_id":"m-1","message":{"text":"привет"}}'

# Ожидается immediately 200 {"data":{"status":"accepted"}} — fast-ack проверка (H3).
# Асинхронно в логах backend появится строка max.send.buttons (dry-run).

# 3) то же сообщение снова (с тем же message_id) — должно быть дедуплицировано:
curl -s -XPOST ... -d '{"chat_id":"c1","message_id":"m-1","message":{"text":"привет"}}'
# Ожидается 200 {"data":{"status":"duplicate"}} — проверка idempotency (H4).

# 4) phase через Finite-state machine (бизнес-флоу лида):
for text in "записаться" "Анна" "+7 916 1234567" "передние тормоза" "скрип" ; do
  curl -s -XPOST ... -d "{\"chat_id\":\"c1\",\"message_id\":\"m-$text\",\"message\":{\"text\":\"$text\"}}"
done
# последний ответ с "accepted"; в БД в таблице leads должна появиться одна запись
#   channel=max, name=Анна, answer={"service":"передние тормоза","symptom":"скрип"}.
```

Проверяемые H-фиксы: H2 (HMAC — в dev подпись не проверяется, в prod проверяется),
H3 (fast 200 до AI), H4 (idempotency по message_id), PII-safe логирование.

## 4. Контракт MAX Bot API — stub-to-real readiness

Проверено без боевых секретов:

- ✅ Интерфейс `max.Client` (`SendMessage`, `SendMessageWithButtons`, `ChatDeepLink`) стабилен.
- ✅ `VerifyWebhook` корректно HMAC-SHA256 (unit-тестировано).
- ✅ `VerifyRequest` читает `X-Max-Bot-Api-Secret` (primary) и `X-Max-Signature` (legacy alias).
- ✅ `ParseWebhook` нормализует поля (`chat_id`, `from.user_id`, `message.text`, `message_id` как string/numeric).
- ⚠️ Реализация боевога `httpClient` (HTTP POST в MAX) — **требует `MAX_API_TOKEN`**. Сейчас `stubClient` только логирует. Инструкция по подключению — `REQUIRES_DECISION.md` RD-9.

## 5. Tests not executed / unchecked

- ❌ **`docker compose -f deployments/docker-compose.yml up --build`** — Docker недоступен в sandbox; валидация статическая. Запустить после relocation.
- ❌ **`npm run build` (frontend)** — Node недоступен в sandbox; валидация только синтаксис Vite `vite.config.js`/`package.json`. Запустить локально.
- ❌ **Интеграционный тест против реальной PostgreSQL 16** — миграции сверенs по SQL, но не запускались на боевой БД. Запустить `docker compose up migrate` первым шагом.
- ❌ **E2E MAX Bot API** — требует боевых секретов (RD-9).
- ❌ Persistence `recentSet` eviction не true-LRU;萍. хватает для single-instance, для multi-instance см. RD-3.

## 6. Test commands cheatsheet

```bash
# Backend
cd backend
go test ./...                          # all
go test -race ./...                    # with race detector
go test -v ./internal/integrations/max # MAX webhook mocks
go vet ./...                           # static checks
gofmt -l internal cmd pkg              # formatting check

# Coverage focus
go test -cover ./internal/priceimport/...
go test -cover ./internal/integrations/max/...

# Frontend (when Node available)
cd frontend
npm install
npm run build                          # build production bundle
```

## 7. Required real MAX credentials (for production verification)

To enable the live integration (RD-9), the operator must provision:

1. **`MAX_API_TOKEN`** — bot token issued by MAX (`@BotMaster`) after bot registration.
2. **`MAX_WEBHOOK_SECRET`** — shared secret configured in MAX Bot API cabinet for webhook signing; must equal server's `MAX_WEBHOOK_SECRET`.
3. **`MAX_NOTIFY_CHAT_ID`** — admin chat ID where new leads/appointments are forwarded.
4. **MAX webhook URL** — point MAX Bot API cabinet to `https://<your-domain>/api/v1/max/webhook`.

Once provisioned: set env, set `MAX_ENABLED=true`, then verify:
- `POST /api/v1/max/webhook` with real MAX-signed payloads is accepted (200) and processed.
- New leда sent to MAX → forwarded to admin chat via real `httpClient` (RD-9 todo).

Без этих секретов остаётся рабочим: web/UI, админка, импорт/экспорт прайса,
база знаний AI, MAX **stub/dry-run** (логи с `enabled=false`).