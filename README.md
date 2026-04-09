# Task Service

Реализация тестового задания для компании **Medods**.

**Автор:** Давиденко Дмитрий Николаевич.

Сервис предоставляет HTTP API на Go для управления обычными задачами и правилами периодичности.

## Что умеет сервис

Сервис работает с двумя основными сущностями:

- **Task** — конкретный экземпляр задачи на определённую дату и время;
- **Task recurrence** — правило периодичности, по которому создаются будущие экземпляры задач.

Поддерживаемые типы периодичности:

- ежедневные задачи — каждый `n`-й день;
- ежемесячные задачи — на конкретное число месяца от `1` до `30`;
- задачи на конкретные даты;
- задачи на нечётные дни месяца;
- задачи на чётные дни месяца.

## Основные проектные решения

В проекте используется следующее разделение ответственности:

- таблица `tasks` хранит реальные задачи, с которыми работает пользователь;
- таблица `task_recurrences` хранит правила генерации;
- таблица `task_recurrence_dates` хранит список дат для режима `specific_dates`.

Такое разделение позволяет:

- не смешивать шаблон и фактическую задачу в одной записи;
- хранить историю уже созданных и выполненных задач;
- изменять правила генерации только для будущих невыполненных экземпляров;
- избегать дублей при повторной материализации задач.

## Поведение периодических задач

### Создание правила периодичности

При создании правила сервис:

1. сохраняет правило в `task_recurrences`;
2. при типе `specific_dates` сохраняет связанные даты в `task_recurrence_dates`;
3. сразу материализует задачи на горизонт планирования вперёд.

Сейчас горизонт планирования составляет **60 дней**.

Все автоматически созданные экземпляры задач создаются со статусом `new`.

### Обновление правила периодичности

При обновлении правила сервис:

1. обновляет запись правила;
2. при необходимости заменяет список `specific_dates`;
3. удаляет будущие невыполненные экземпляры этой серии;
4. заново материализует задачи на текущий горизонт планирования.

Прошлые и уже завершённые задачи не удаляются.

### Удаление правила периодичности

При удалении правила сервис:

1. удаляет будущие невыполненные экземпляры серии;
2. удаляет само правило.

Исторические и завершённые задачи сохраняются. После удаления правила их `recurrence_id` становится `NULL` за счёт `ON DELETE SET NULL`.

### Актуализация будущих задач

При вызове `GET /api/v1/tasks` сервис перед возвратом списка задач синхронизирует будущие экземпляры для всех активных правил периодичности. Это позволяет поддерживать актуальный список задач без отдельного cron или background worker.

## Обычные задачи

Обычная задача создаётся через `/api/v1/tasks`.

Поле `scheduled_at` можно:

- передать явно;
- не передавать, тогда сервис подставляет текущее UTC-время.

Обычные задачи не требуют создания правила периодичности.

## Требования

- Go `1.23+`
- Docker и Docker Compose

## Быстрый запуск через Docker Compose

```bash
docker compose up --build
```

После запуска сервис доступен по адресу:

```text
http://localhost:8000
```

## Сброс базы данных

Если `postgres` уже запускался ранее и нужно поднять проект на чистой схеме, пересоздай volume:

```bash
docker compose down -v
docker compose up --build
```

SQL-файлы из директории `migrations/` монтируются в `docker-entrypoint-initdb.d` и применяются только при инициализации нового data volume.

## Swagger

Swagger UI:

```text
http://localhost:8000/swagger/
```

OpenAPI JSON:

```text
http://localhost:8000/swagger/openapi.json
```

## Структура API

Базовый префикс API:

```text
/api/v1
```

### Обычные задачи

- `POST /api/v1/tasks`
- `GET /api/v1/tasks`
- `GET /api/v1/tasks/{id}`
- `PUT /api/v1/tasks/{id}`
- `DELETE /api/v1/tasks/{id}`

### Правила периодичности

- `POST /api/v1/task-recurrences`
- `GET /api/v1/task-recurrences`
- `GET /api/v1/task-recurrences/{id}`
- `PUT /api/v1/task-recurrences/{id}`
- `DELETE /api/v1/task-recurrences/{id}`

## Фильтры списка задач

`GET /api/v1/tasks` поддерживает query-параметры:

- `scheduled_from` — нижняя граница по `scheduled_at` в формате RFC3339;
- `scheduled_to` — верхняя граница по `scheduled_at` в формате RFC3339;
- `status` — фильтр по статусу: `new`, `in_progress`, `done`.

Пример:

```text
/api/v1/tasks?scheduled_from=2026-04-01T00:00:00Z&scheduled_to=2026-04-30T23:59:59Z&status=new
```

## Формат задач

### Task

```json
{
  "id": 1,
  "title": "Prepare release",
  "description": "Collect release notes and check migrations",
  "status": "new",
  "scheduled_at": "2026-04-10T09:00:00Z",
  "recurrence_id": 7,
  "created_at": "2026-04-09T12:00:00Z",
  "updated_at": "2026-04-09T12:00:00Z"
}
```

Поле `recurrence_id`:

- содержит идентификатор правила, если задача создана из серии;
- отсутствует или равно `null` для обычной задачи.

## Формат правил периодичности

### Task recurrence

```json
{
  "id": 7,
  "title": "Daily patient follow-up",
  "description": "Call patients after discharge",
  "type": "daily_every_n_days",
  "starts_at": "2026-04-10T09:00:00Z",
  "ends_on": "2026-06-10T09:00:00Z",
  "interval_days": 2,
  "day_of_month": null,
  "specific_dates": null,
  "created_at": "2026-04-09T12:00:00Z",
  "updated_at": "2026-04-09T12:00:00Z"
}
```

### Поддерживаемые значения `type`

- `daily_every_n_days`
- `monthly_day_of_month`
- `specific_dates`
- `odd_days_of_month`
- `even_days_of_month`

## Правила валидации для периодичности

### Для всех правил

- `title` обязателен;
- `starts_at` обязателен;
- `ends_on`, если передан, не может быть раньше `starts_at`;
- тип периодичности должен быть валидным.

### Для `daily_every_n_days`

- `interval_days` обязателен;
- `interval_days >= 1`;
- `day_of_month` не допускается;
- `specific_dates` не допускается.

### Для `monthly_day_of_month`

- `day_of_month` обязателен;
- `day_of_month` должен быть в диапазоне `1..30`;
- `interval_days` не допускается;
- `specific_dates` не допускается.

### Для `specific_dates`

- `specific_dates` обязателен и не должен быть пустым;
- значения должны быть уникальными;
- формат даты: `YYYY-MM-DD`;
- `interval_days` не допускается;
- `day_of_month` не допускается.

### Для `odd_days_of_month` и `even_days_of_month`

- `interval_days` не допускается;
- `day_of_month` не допускается;
- `specific_dates` не допускается.

## Примеры запросов

### Создать обычную задачу

```json
{
  "title": "Prepare release",
  "description": "Collect release notes and check migrations",
  "status": "new",
  "scheduled_at": "2026-04-10T09:00:00Z"
}
```

### Создать правило daily every N days

```json
{
  "title": "Daily patient follow-up",
  "description": "Call patients after discharge",
  "type": "daily_every_n_days",
  "starts_at": "2026-04-10T09:00:00Z",
  "interval_days": 2
}
```

### Создать правило monthly day of month

```json
{
  "title": "Monthly report",
  "description": "Prepare monthly report",
  "type": "monthly_day_of_month",
  "starts_at": "2026-04-10T09:00:00Z",
  "day_of_month": 15
}
```

### Создать правило specific dates

```json
{
  "title": "Specific campaign dates",
  "description": "Special campaign task dates",
  "type": "specific_dates",
  "starts_at": "2026-04-10T09:00:00Z",
  "specific_dates": ["2026-04-10", "2026-04-15", "2026-04-20"]
}
```

### Создать правило odd days

```json
{
  "title": "Odd day task",
  "description": "Task for odd days of month",
  "type": "odd_days_of_month",
  "starts_at": "2026-04-10T09:00:00Z"
}
```

### Создать правило even days

```json
{
  "title": "Even day task",
  "description": "Task for even days of month",
  "type": "even_days_of_month",
  "starts_at": "2026-04-10T09:00:00Z"
}
```

## Схема базы данных

### `tasks`

Хранит экземпляры задач.

Основные поля:

- `id`
- `title`
- `description`
- `status`
- `scheduled_at`
- `recurrence_id`
- `created_at`
- `updated_at`

### `task_recurrences`

Хранит правила периодичности.

Основные поля:

- `id`
- `title`
- `description`
- `type`
- `starts_at`
- `ends_on`
- `interval_days`
- `day_of_month`
- `created_at`
- `updated_at`

### `task_recurrence_dates`

Хранит даты для режима `specific_dates`.

Поля:

- `recurrence_id`
- `occurrence_date`

## Индексы и ограничения

В базе настроены:

- индекс по `tasks.status`;
- индекс по `tasks.scheduled_at`;
- индекс по `tasks.recurrence_id`;
- уникальный индекс по `(recurrence_id, scheduled_at)` для защиты от дублей задач внутри одной серии;
- ограничения на допустимые типы периодичности и допустимые сочетания полей в `task_recurrences`.

## Поведение, которое важно учитывать

- повторный вызов `GET /tasks` не должен создавать дубли задач одной серии;
- завершённые задачи (`status = done`) сохраняются при обновлении и удалении правил;
- будущие невыполненные задачи серии пересобираются на основе актуального правила;
- `specific_dates` в HTTP-контракте передаются и возвращаются в формате `YYYY-MM-DD`;
- фильтрация списка задач по диапазону и статусу выполняется через query-параметры.
