-- 0001_init.up.sql
-- Timemachine auto service: initial schema
-- All monetary values stored as integer cents to avoid float rounding.

CREATE EXTENSION IF NOT EXISTS "pgcrypto";

CREATE TABLE service_categories (
    id          bigint PRIMARY KEY GENERATED ALWAYS AS IDENTITY,
    name        text NOT NULL,
    slug        text NOT NULL UNIQUE,
    parent_id   bigint REFERENCES service_categories(id) ON DELETE SET NULL,
    sort_order  integer NOT NULL DEFAULT 0,
    created_at  timestamptz NOT NULL DEFAULT now()
);
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- pg_trgm powers the optional trigram search index on services.name.
-- Best effort: if the extension can't be installed (no superuser in the
-- role), the trigram index is skipped — the ilike search still works.
CREATE EXTENSION IF NOT EXISTS pg_trgm;

CREATE TABLE service_categories (
    id                 bigint PRIMARY KEY GENERATED ALWAYS AS IDENTITY,
    category_id        bigint NOT NULL REFERENCES service_categories(id) ON DELETE RESTRICT,
    name               text NOT NULL,
    description        text NOT NULL DEFAULT '',
    long_description   text NOT NULL DEFAULT '',
    price_cents        bigint NOT NULL CHECK (price_cents >= 0),
    currency           text NOT NULL DEFAULT 'RUB' CHECK (currency IN ('RUB','USD','EUR')),
    duration_minutes   integer NOT NULL DEFAULT 60 CHECK (duration_minutes > 0),
    is_from_price      boolean NOT NULL DEFAULT false,
    is_active          boolean NOT NULL DEFAULT true,
    archived           boolean NOT NULL DEFAULT false,
    sort_order         integer NOT NULL DEFAULT 0,
    created_at         timestamptz NOT NULL DEFAULT now(),
    updated_at         timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX ix_services_category ON services(category_id);
CREATE INDEX ix_services_active ON services(is_active) WHERE is_active = true;
CREATE INDEX IF NOT EXISTS ix_services_name_trgm ON services USING gin (name gin_trgm_ops);

-- users / admins
CREATE TABLE users (
    id            bigint PRIMARY KEY GENERATED ALWAYS AS IDENTITY,
    email         text NOT NULL UNIQUE,
    password_hash text NOT NULL,
    role          text NOT NULL DEFAULT 'manager' CHECK (role IN ('admin','manager')),
    name          text NOT NULL DEFAULT '',
    is_active     boolean NOT NULL DEFAULT true,
    created_at    timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE price_import_jobs (
    id          bigint PRIMARY KEY GENERATED ALWAYS AS IDENTITY,
    status      text NOT NULL DEFAULT 'pending'
                CHECK (status IN ('pending','validated','applied','failed','canceled')),
    file_name   text NOT NULL DEFAULT '',
    total_rows  integer NOT NULL DEFAULT 0,
    added       integer NOT NULL DEFAULT 0,
    updated     integer NOT NULL DEFAULT 0,
    skipped     integer NOT NULL DEFAULT 0,
    errors      jsonb NOT NULL DEFAULT '[]'::jsonb,
    created_by  bigint REFERENCES users(id) ON DELETE SET NULL,
    created_at  timestamptz NOT NULL DEFAULT now(),
    applied_at  timestamptz
);

CREATE TABLE price_import_rows (
    id          bigint PRIMARY KEY GENERATED ALWAYS AS IDENTITY,
    job_id      bigint NOT NULL REFERENCES price_import_jobs(id) ON DELETE CASCADE,
    row_number  integer NOT NULL,
    action      text NOT NULL CHECK (action IN ('create','update','skip','error')),
    service_id  bigint REFERENCES services(id) ON DELETE SET NULL,
    payload     jsonb NOT NULL DEFAULT '{}'::jsonb,
    errors      jsonb NOT NULL DEFAULT '[]'::jsonb,
    UNIQUE (job_id, row_number)
);
CREATE INDEX ix_import_rows_job ON price_import_rows(job_id);

CREATE TABLE appointments (
    id            bigint PRIMARY KEY GENERATED ALWAYS AS IDENTITY,
    name          text NOT NULL,
    phone         text NOT NULL,
    car_make      text NOT NULL DEFAULT '',
    car_model     text NOT NULL DEFAULT '',
    gov_number    text NOT NULL DEFAULT '',
    vin           text NOT NULL DEFAULT '',
    service_id    bigint REFERENCES services(id) ON DELETE SET NULL,
    desired_at    timestamptz,
    comment       text NOT NULL DEFAULT '',
    status        text NOT NULL DEFAULT 'new'
                  CHECK (status IN ('new','confirmed','canceled','done','no_show')),
    manager_note  text NOT NULL DEFAULT '',
    source        text NOT NULL DEFAULT 'site',
    external_ref  text NOT NULL DEFAULT '',
    created_at    timestamptz NOT NULL DEFAULT now(),
    updated_at    timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX ix_appointments_status ON appointments(status);
CREATE INDEX ix_appointments_created ON appointments(created_at);

CREATE TABLE leads (
    id             bigint PRIMARY KEY GENERATED ALWAYS AS IDENTITY,
    external_ref   text NOT NULL DEFAULT '',
    channel        text NOT NULL CHECK (channel IN ('site','max')),
    name           text NOT NULL DEFAULT '',
    phone          text NOT NULL DEFAULT '',
    max_chat_id    text NOT NULL DEFAULT '',
    answer         jsonb NOT NULL DEFAULT '{}'::jsonb,
    status         text NOT NULL DEFAULT 'raw'
                   CHECK (status IN ('raw','qualified','handled','trash')),
    assigned_to   bigint REFERENCES users(id) ON DELETE SET NULL,
    appointment_id bigint REFERENCES appointments(id) ON DELETE SET NULL,
    created_at     timestamptz NOT NULL DEFAULT now(),
    handled_at     timestamptz
);
CREATE INDEX ix_leads_status ON leads(status);
CREATE INDEX ix_leads_channel ON leads(channel);

CREATE TABLE faq_items (
    id            bigint PRIMARY KEY GENERATED ALWAYS AS IDENTITY,
    question      text NOT NULL,
    answer        text NOT NULL,
    category      text NOT NULL DEFAULT '',
    sort_order    integer NOT NULL DEFAULT 0,
    is_published boolean NOT NULL DEFAULT true,
    created_at    timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE knowledge_items (
    id         bigint PRIMARY KEY GENERATED ALWAYS AS IDENTITY,
    title      text NOT NULL,
    body       text NOT NULL,
    tags       text[] NOT NULL DEFAULT '{}',
    source     text NOT NULL DEFAULT '',
    updated_by bigint REFERENCES users(id) ON DELETE SET NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE reviews (
    id          bigint PRIMARY KEY GENERATED ALWAYS AS IDENTITY,
    author      text NOT NULL,
    rating       smallint NOT NULL CHECK (rating BETWEEN 1 AND 5),
    body        text NOT NULL DEFAULT '',
    car_info    text NOT NULL DEFAULT '',
    is_published boolean NOT NULL DEFAULT true,
    created_at  timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE max_messages (
    id         bigint PRIMARY KEY GENERATED ALWAYS AS IDENTITY,
    chat_id    text NOT NULL,
    direction  text NOT NULL CHECK (direction IN ('in','out')),
    text       text NOT NULL DEFAULT '',
    payload    jsonb NOT NULL DEFAULT '{}'::jsonb,
    session_id bigint,
    created_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX ix_max_chat ON max_messages(chat_id, created_at);

CREATE TABLE sessions (
    id         bigint PRIMARY KEY GENERATED ALWAYS AS IDENTITY,
    chat_id    text NOT NULL,
    scope      text NOT NULL DEFAULT 'max' CHECK (scope IN ('max','web')),
    state      jsonb NOT NULL DEFAULT '{}'::jsonb,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX ix_sessions_chat ON sessions(chat_id);

CREATE TABLE audit_logs (
    id         bigint PRIMARY KEY GENERATED ALWAYS AS IDENTITY,
    actor_id   bigint REFERENCES users(id) ON DELETE SET NULL,
    action     text NOT NULL,
    entity     text NOT NULL,
    entity_id  text NOT NULL DEFAULT '',
    diff       jsonb NOT NULL DEFAULT '{}'::jsonb,
    created_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX ix_audit_entity ON audit_logs(entity, entity_id);
CREATE INDEX ix_audit_actor ON audit_logs(actor_id);

-- updated_at trigger for services/appointments
CREATE OR REPLACE FUNCTION touch_updated_at() RETURNS trigger AS $$
BEGIN
    NEW.updated_at = now();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER services_touch BEFORE UPDATE ON services
    FOR EACH ROW EXECUTE FUNCTION touch_updated_at();

CREATE TRIGGER appointments_touch BEFORE UPDATE ON appointments
    FOR EACH ROW EXECUTE FUNCTION touch_updated_at();