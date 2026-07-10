# БД-схема «Машина времени»

Хранилище: PostgreSQL 16. Денежные суммы — целые копейки (`price_cents`,
`bigint`) против ошибок округления.

## Сущности и связи

```
service_categories 1───* services
users (admins)            price_import_jobs *──1 users (created_by)
                          price_import_rows  *──1 price_import_jobs (cascade)
                          price_import_rows  *──1 services (service_id)
appointments              services 1──0..1 appointments (service_id)
leads   *──1 users (assigned_to)
leads   *──1 appointments (appointment_id)
faq_items
knowledge_items (updated_by → users)
reviews
max_messages
sessions (chat state)
audit_logs (actor_id → users)
```

## Таблицы

### service_categories
`id`, `name`, `slug(unique)`, `parent_id?(self FK)`, `sort_order`,
`created_at`

### services
`id`, `category_id(FK)`, `name`, `description`, `long_description`,
`price_cents(n>=0)`, `currency(RUB|USD|EUR)`, `duration_minutes(n>0)`,
`is_from_price`, `is_active`, `archived`, `sort_order`, `created_at`,
`updated_at` (trigger). Индексы: by category, by active, gin trgm по name.

### users (админы)
`id`, `email(unique)`, `password_hash (bcrypt)`, `role(admin|manager)`,
`name`, `is_active`, `created_at`

### price_import_jobs
`id`, `status(pending|validated|applied|failed|canceled)`, `file_name`,
`total_rows`, `added`, `updated`, `skipped`, `errors jsonb`, `created_by`,
`created_at`, `applied_at`

### price_import_rows
`id`, `job_id(FK cascade)`, `row_number`, `action(create|update|skip|error)`,
`service_id?`, `payload jsonb`, `errors jsonb`. `UNIQUE(job_id,row_number)`.

### appointments
`id`, `name`, `phone`, `car_make`, `car_model`, `gov_number`, `vin`,
`service_id?`, `desired_at`, `comment`, `status(new|confirmed|canceled
|done|no_show)`, `manager_note`, `source`, `external_ref`,
`created_at`, `updated_at(trigger)`

### leads
`id`, `external_ref`, `channel(site|max)`, `name`, `phone`, `max_chat_id`,
`answer jsonb`, `status(raw|qualified|handled|trash)`, `assigned_to?`,
`appointment_id?`, `created_at`, `handled_at?`

### faq_items
`id`, `question`, `answer`, `category`, `sort_order`, `is_published`,
`created_at`

### knowledge_items
`id`, `title`, `body`, `tags text[]`, `source`, `updated_by?`,
`created_at`, `updated_at`

### reviews
`id`, `author`, `rating(1..5)`, `body`, `car_info`, `is_published`,
`created_at`

### max_messages
`id`, `chat_id`, `direction(in|out)`, `text`, `payload jsonb`, `session_id?`,
`created_at`. Индекс по `(chat_id, created_at)` для истории диалога.

### sessions
`id`, `chat_id`, `scope(max|web)`, `state jsonb`, `created_at`,
`updated_at`. Хранит состояние чат-бота между шагами.

### audit_logs
`id`, `actor_id?`, `action`, `entity`, `entity_id`, `diff jsonb`,
`created_at`

## Миграции
Формат golang-migrate: `NNNN_name.up.sql` / `NNNN_name.down.sql` в
`backend/internal/migrations`. Применяются CLI `cmd/migrate` (embed FS).