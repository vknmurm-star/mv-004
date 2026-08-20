# «На высоте» — контентный журнал для мужчин 40+

Next.js 16 App Router сайт-журнал о здоровье, стиле и самочувствии для мужчин 40+. Тон проекта: спокойный, уважительный, без покровительственности, «бро»-сленга и медицинских назначений.

## Айдентика

Основная выбранная палитра: **тёплый графит + медь + светлый песок**. Она выглядит маскулинно, но не мрачно: графит даёт собранность, медь — характер, песочный фон — воздух и редакционную мягкость.

Альтернативы для будущего редизайна:

1. **Тёмно-синий + горчичный + молочный** — более журнальный и деловой вариант.
2. **Оливковый + сталь + тёплый серый** — спокойнее, ближе к outdoor/wellbeing.
3. **Графит + медь + песок** — текущий вариант.

Заголовочный шрифт: **Fraunces** — выразительный, зрелый, с характером, но без декоративности Cormorant Garamond.

## Технологии

- Next.js 16, App Router
- React 19
- Tailwind CSS
- MDX-статьи с frontmatter
- Decap CMS по адресу `/admin`
- JSON-LD: `Article`, `BreadcrumbList`, `CollectionPage`
- `sitemap.xml` и `robots.txt` через App Router metadata routes

## Структура проекта

```text
frontend/
├─ app/                    # App Router страницы, metadata, sitemap, robots
│  ├─ articles/[slug]/     # Страница статьи
│  ├─ category/[slug]/     # Страница рубрики
│  ├─ globals.css          # Tailwind и дизайн-токены
│  ├─ layout.jsx           # Общий layout, header/footer, базовые meta
│  └─ page.jsx             # Главная
├─ components/             # Карточки статей, JSON-LD, блок «Читайте также»
├─ content/articles/       # Стартовые MDX-статьи
├─ lib/content.js          # Чтение MDX, категории, JSON-LD helpers
├─ public/admin/           # Decap CMS config и admin shell
├─ public/images/articles/ # SVG-плейсхолдеры обложек
├─ Dockerfile
├─ next.config.mjs
├─ package.json
└─ postcss.config.mjs
```

## Категории и стартовый контент

На старте добавлены 4 категории по 2 статьи:

- **Здоровье и энергия** — сон, восстановление, профилактические чекапы без диагнозов.
- **Стиль** — гардероб и уход за собой без «домохозяйской» подачи.
- **Форма и тело** — тренировки 40+, осанка, подвижность.
- **Отношения и уверенность** — карьера, семья, внутренний тон без клише о кризисе среднего возраста.

Формат frontmatter статьи:

```yaml
title: "Заголовок"
description: "Краткое описание для SEO и карточек"
date: "2026-08-01"
category: "health-energy"
coverImage: "/images/articles/sleep-checkup.svg"
coverImageAlt: "Описание изображения"
```

## Локальный запуск

```bash
cd frontend
npm install
npm run dev
```

Сайт будет доступен на `http://localhost:3000`.

## Production build

```bash
cd frontend
npm install
npm run build
npm run start
```

## Docker

```bash
cd frontend
docker build -t na-vysote .
docker run --rm -p 3000:3000 -e NEXT_PUBLIC_SITE_URL=http://localhost:3000 na-vysote
```

## Decap CMS

Админка доступна по адресу:

```text
/admin
```

Для локальной работы с Decap CMS можно запустить local backend:

```bash
cd frontend
npx decap-server
npm run dev
```

Конфигурация Decap CMS лежит в `frontend/public/admin/config.yml`. Новые статьи сохраняются в `frontend/content/articles` как `.mdx` с frontmatter.

## SEO/GEO

- На главной, рубриках и статьях настроены уникальные title/description/canonical/OG/Twitter metadata.
- Для статей генерируется JSON-LD `Article`.
- Для рубрик и главной генерируется JSON-LD `CollectionPage`.
- Для навигационной цепочки генерируется `BreadcrumbList`.
- `robots.txt` разрешает обычных поисковых роботов и GEO-краулеров: `GPTBot`, `PerplexityBot`, `ClaudeBot`.
- `sitemap.xml` включает главную, категории и все статьи.

## Изображения

Пока используются SVG-плейсхолдеры в `frontend/public/images/articles`. Реальные фотографии можно заменить вручную, обновив `coverImage` и `coverImageAlt` в frontmatter статьи.
