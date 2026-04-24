# Task Service

Сервис для управления задачами с HTTP API на Go.

## Требования

- Go `1.23+`
- Docker и Docker Compose

## Быстрый запуск через Docker Compose

```bash
docker compose up --build
```

После запуска сервис будет доступен по адресу `http://localhost:8080`.

Если `postgres` уже запускался ранее со старой схемой, пересоздай volume:

```bash
docker compose down -v
docker compose up --build
```

Причина в том, что SQL-файл из `migrations/0001_create_tasks.up.sql` монтируется в `docker-entrypoint-initdb.d` и применяется только при инициализации пустого data volume.

## Swagger

Swagger UI:

```text
http://localhost:8080/swagger/
```

OpenAPI JSON:

```text
http://localhost:8080/swagger/openapi.json
```

## API

Базовый префикс API:

```text
/api/v1
```

Основные маршруты:

- `POST /api/v1/tasks`
- `GET /api/v1/tasks`
- `GET /api/v1/tasks/{id}`
- `PUT /api/v1/tasks/{id}`
- `DELETE /api/v1/tasks/{id}`

## Повторяющиеся задачи

При создании задачи можно передать поля:

- `recurrence_type`: `daily`, `monthly`, `dates`, `even_odd`
- `recurrence_config`: JSON-конфиг для выбранного типа

Если у задачи задана периодичность, она сохраняется как шаблон, а фоновый воркер
периодически создаёт экземпляры задач и сдвигает `next_run_date`.

## Что реализовано по тестовому заданию

- Добавлены настройки периодичности: `daily`, `monthly`, `dates`, `even_odd`.
- Реализована валидация `recurrence_type` и `recurrence_config` при создании.
- Периодическая задача сохраняется как шаблон (запись с `parent_task_id = NULL`).
- Фоновый воркер создаёт экземпляры задач из шаблонов по `next_run_date`.
- После создания экземпляра воркер пересчитывает следующую дату запуска шаблона.
- Добавлен фильтр в `GET /api/v1/tasks?filter=`:
	- `all` (по умолчанию),
	- `templates`,
	- `instances`.

## Принятые допущения

- Шаблон задачи хранится в той же таблице `tasks`.
- Экземпляр задачи - отдельная запись со ссылкой `parent_task_id` на шаблон.
- Для `monthly` допустимы дни месяца от `1` до `30` (согласно формулировке задания).
- Для `dates` при отсутствии будущих дат новые экземпляры больше не создаются.
- Время расчётов и запусков - `UTC`.

## Примеры запросов

Создать ежедневный шаблон:

```json
{
	"title": "Ежедневный обзвон",
	"status": "new",
	"recurrence_type": "daily",
	"recurrence_config": {
		"interval_days": 1
	}
}
```

Получить только шаблоны:

```text
GET /api/v1/tasks?filter=templates
```

Получить только экземпляры:

```text
GET /api/v1/tasks?filter=instances
```
