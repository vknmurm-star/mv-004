# REQUIRES_DECISION — Машина времени

Список спорных решений, обнаруженных при аудите, которые **не** исправлены
автоматически, потому что требуют продукт/архитектурного выбора владельцем.
Каждый пункт: контекст, варианты, рекомендация исполнителя аудита.

---

## RD-1. RBAC: `manager` против `admin`
- **Контекст:** `middleware.RequireRole`_exists (`internal/middleware/auth.go:63`) и в `users.role` хранятся значения `'admin'` и `'manager'`, но роутер (`router.go`) не применяет `RequireRole` ни на одном эндпоинте. Все admin-эндпоинты доступны любому аутентифицированному пользователю с любой ролью.
- **Варианты:**
  - A) Оставить как есть (обе роли = доверенный менеджер, продуктовое решение).
  - B) Разделить: `DELETE service`, `apply import job`, `delete knowledge` — только `admin`; остальное — `manager`+`admin`.
  - C) Ввести роли `viewer` (только чтение) для showcases.
- **Рекомендация:** B. Минимально-достаточное разделение: опасные операции — admin; операции просмотра/правки — manager.
- **Где править:** `router.go` — обернуть `r.With(middleware.RequireRole("admin"))` вокруг критичных эндпоинтов.

## RD-2. `chi.middleware.RealIP` без `TrustedNetworks`
- **Контекст:** `router.go:53` подключает `chimw.RealIP`, который по умолчанию доверяет любому `X-Forwarded-For`. После hardening `clientIP` использует только `r.RemoteAddr` (rate-limit корректен), но сам `RealIP` всё ещё принимает XFF от любого источника → `r.RemoteAddr` в handler'ах может доверять подделанному IP.
- **Варианты:**
  - A) Убрать `chimw.RealIP` и доверять только `r.RemoteAddr` от L4 (TCP peer IP).
  - B) Настроить chi/v5 RealIP с TrustedNetworks (CIDR вашего reverse proxy / CDN).
  - C) Переехать на сторонний `ip-from-trusted-proxy` пакет.
- **Рекомендация:** B. Для прод-деплоя за Nginx/Cloudflare — задать TrustedNetworks explicit.
- **Где править:** `router.go`, конфиг `TRUSTED_PROXY_CIDRS` env.

## RD-3. `seatStore` in-memory — не масштабируется горизонтально
- **Контекст:** `httpserver.support.go::seatStore` хранит состояние диалога в памяти процесса. В `migrations/0001` уже подготовлена таблица `sessions` (state jsonb), но agent её не использует.
- **Варианты:**
  - A) Оставить для single-instance MVP.
  - B) Реализовать `sessionsRepo` + периодическая очистка старых сидов; агент читает/пишет через него.
  - C) Использовать Redis (как опциональный компонент в compose) для ephemeral state.
- **Рекомендация:** B — минимальное усилие, использует уже готовую таблицу, работает в одном docker-compose (один backend-контейнер) и не требует новой инфры.
- **Где править:** новый `domain/session/repo.go`, `aiagent.Seat` обёрнут в repo, хендлер `processMaxMessage` грузит/сохраняет через репо.

## RD-4. Дублирующиеся колонки в CSV-импорте принимаются
- **Контекст:** `priceimport.mapHeader` перезаписывает индексы при дублях (последняя колонка побеждает). Менеджер, склеивший два столбца `name` в рабочем файле, получит тихое «перетирание» без предупреждения.
- **Варианты:**
  - A) Оставить как есть, фикс в docs.
  - B) Возвращать ошибку `duplicate column "name"` если два канонических столбца найдены.
  - C) Показывать warning в `summary.errors` без фейла всего файла.
- **Рекомендация:** B — строгая валидацияHeaders; `mapHeader` возвращает `(idx, duplicateColumns)`, хендлер репортит.
- **Где править:** `priceimport/csv.go`, `handlers_admin.go::AdminImportPrices` отдаёт duplicated в meta.

## RD-5. `catLookup.Resolve` использует `context.Background()`
- **Контекст:** `priceimport/service.go::catLookup.Resolve` создаёт `context.WithTimeout(context.Background(), 5s)`, игнорируя родительский ctx. При отмене операции импорта (клиент закрыл соединение)Cat lookup всё ещё нагружает БД.
- **Варианты:**
  - A) Оставить (cat-lookup тривиален, 5s фиксированный timeout).
  - B) Передавать `ctx context.Context` в `categoryResolver.Resolve`, деррivать от него.
- **Рекомендация:** B. Меняет интерфейс `categoryResolver` — простой рефактор. Мелкая, но безопасная правка; отложена только потому, что трогает интерфейс который задействован в `csv.go`/`xlsx.go`/тестах.
- **Где править:** `priceimport/csv.go:69-72` + `xlsx.go` + тесты.

## RD-6. JWT в localStorage (XSS-риск)
- **Контекст:** `frontend/src/utils/auth.jsx` хранит JWT в `localStorage`. Любой成功的 XSS раскрывает токен. Стандартный tradeoff для SPA admin panel.
- **Варианты:**
  - A) Оставить (текущий XSS-mitigation: CSP `default-src 'self'; frame-ancestors 'none'` в prod).
  - B) Перейти на httpOnly cookie + CSRF-токены.
- **Рекомендация:** B для прод- MVP, но требует backend изменение (Cookie → CSRF). Великовато для safe fix.
- **Где править:** backend `POST /admin/login` пишет httpOnly Cookie; middleware `RequireAuth` читает Cookie OR Bearer; `csrf` middleware.

## RD-7. `/api/v1/reviews` публичный POST без модерации-гейта
- **Контекст:** Любой может создать отзыв (`is_published=false` по умолчанию → надо модерировать вручную). Нет капчи/hCaptcha. Только rate-limit (10/час per IP).
- **Варианты:**
  - A) Оставить (moderation на стороне админа, rate-limit терпит).
  - B) hCaptcha/Turnstile на этом и `CreateAppointment`.
- **Рекомендация:** B для прод (защита от спама-флуда), но требует введения env `HCAPTCHA_SECRET`.
- **Где править:** новая `middleware.Captcha`, проверка siteverify на `CreateReview`/`CreateAppointment`.

## RD-8. `frontend/api.js` не обрабатывает 401 (токен-устаревание)
- **Контекст:** `frontend/src/utils/api.js::request` — при 401 кидает Error как при любой другой ошибке; токен остаётся в localStorage, UI продолжает слать запросы с протухшим токеном.
- **Рекомендация:** авто-`clear()` + редирект на `/admin/login` при 401. Тривиальная UX-правка.
- **Почему не в safe fixes:** правка UX-флоу админки, потенциально меняет поведение пользователя. Решать как продукт.

## RD-9. Реальный HTTP-клиент MAX Bot API
- **Контекст:** `integrations/max/client.go` содержит `stubClient`, который только логирует. Интерфейс `Client` готов к замене, но реальный HTTP-вызов (`POST ${MAX_API_BASE}/messages/sendText`, `Authorization: Bearer ${MAX_API_TOKEN}`) не реализован, потому что нет боевого токена.
- **Рекомендация:** после получения боевых `MAX_API_TOKEN` от MAX (`@BotMaster`):
  1. Создать `integrations/max/http_client.go` c `httpClient` реализующим `Client`.
  2. `SendMessage` → `POST {APIBase}/messages/sendText`, body `{"chat_id":..., "text":...}`, header `Authorization: Bearer {APIToken}`.
  3. `SendMessageWithButtons` → эндпоинт inline-кнопок MAX (по их спецификации).
  4. Заменить `BuildClient` селектор: `if cfg.Max.Enabled && cfg.Max.APIToken != "" → httpClient else stubClient`.
- **Где править:** `integrations/max/http_client.go` (new), `integrations/max/notifier.go::BuildClient`.
- **Зона проверки без секретов:** покрыты тестами только verifier (`HMAC`), `ParseWebhook` и stub. Боевой HTTP-клиент помечен как «needs real MAX credentials» в `TEST_PLAN`.

## RD-10. seed admin пароль захардкожен в миграции
- **Контекст:** `migrations/0002_seed.up.sql` вставляет bcrypt-хэш пароля `admin12345`. Любой кто читает репо знает демо-пароль.
- **Рекомендация:** в прод-деплоях:
  - удалить/не применять `0002_seed.up.sql` (migrate `-direction=up` можно ограничить до `0001`+`0003`);
  - или добавить env-driven seed через отдельный `cmd/seed` с `ADMIN_BOOTSTRAP_PASSWORD`.
- **Где править:** новый `cmd/seed/main.go`, `0002_seed.up.sql` — закомментировать административный insert, оставить только demo-услуги.

## RD-11. Observability: метрики/trace/logsstructured
- **Контекст:** Логи — `slog` JSON в stdout. Нет `/metrics` для Prometheus, нет OpenTelemetry-tracing, нет correlation между сайтом→вебхуком→MAX через trace_id.
- **Варианты:** добавить `prometheus` middleware + `/metrics`; OTel HTTP auto-instrumentation.
- **Рекомендация:** следующий эпик после MVP. Помечено здесь, не в safe fixes.

---

## Что нужно от владельца прод-инсталляции

1. **MAX API токен + webhook secret** (`MAX_API_TOKEN`, `MAX_WEBHOOK_SECRET`) — для RD-9.
2. **CORS origins** для реального домена (`CORS_ALLOWED_ORIGINS`).
3. **JWT_SECRET** ≥ 32 символов (в prod конфигурация отвергает короткий).
4. **Trusted proxy CIDRs** для chi `RealIP` (RD-2).
5. **Смена admin-пароля** (RD-10).