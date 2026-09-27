# TripGo — Trip Service

Лабораторная работа 1. HTTP API и PostgreSQL

**Что умеет сервис:**
- создавать поездки (`POST /api/v1/trips`);
- отдавать поездку по id (`GET /api/v1/trips/{tripId}`);
- завершать поездку (`POST /api/v1/trips/{tripId}/finish`);
- сообщать о состоянии процесса и готовности (`GET /health`, `GET /ready`).

**Стек:**
- Go 1.24+
- `net/http` + [`go-chi/chi/v5`](https://github.com/go-chi/chi)
- [`jackc/pgx/v5`](https://github.com/jackc/pgx) (`pgxpool`) - работа с PostgreSQL
- [`Masterminds/squirrel`](https://github.com/Masterminds/squirrel) - SQL-билдер
- [`pressly/goose`](https://github.com/pressly/goose) - миграции
- [`oapi-codegen`](https://github.com/oapi-codegen/oapi-codegen) - кодогенерация из OpenAPI
- `log/slog` - структурированные JSON-логи

Контракты: [`contracts/openapi/trip-service.openapi.yaml`](contracts/openapi/trip-service.openapi.yaml), [`contracts/schema.md`](contracts/schema.md).

---

## Требования

- Linux, macOS или WSL2 (Ubuntu)
- Go 1.24+
- Docker (для локального окружения через `tripgoctl`)
- Утилита `tripgoctl` - установка описана в [course-infra](https://github.com/course-go-autumn-2026/course-infra)

---

## Быстрый старт

### 1. Клонировать репозиторий

```bash
git clone git@github.com:Ksusha-Pushkova/trip_go.git
cd trip_go
```

### 2. Загрузить зависимости

```bash
go mod download
```

### 3. Поднять локальное окружение

```bash
tripgoctl cluster start        
tripgoctl environment start    # поднимет PostgreSQL, создаст .env
```

`environment start` создаёт файл `.env` в корне репозитория с реальными адресами. Этот файл не коммитится (он в `.gitignore`) - у каждого разработчика свой.

### 4. Настроить переменные окружения

Если `.env` уже создан `tripgoctl` - можно использовать его как есть. Иначе - скопировать шаблон:

```bash
cp .env.example .env
```

И подставить свои значения (по умолчанию подходят для локального окружения `tripgoctl`).

### 5. Накатить миграции

```bash
make migrate
```

### 6. Запустить сервис

```bash
make run
```

Сервис слушает `:8080` (адрес из `HTTP_ADDR`).

### 7. Проверить

```bash
curl -i localhost:8080/health
curl -i localhost:8080/ready
```

Оба должны вернуть `200 OK` и `{"status":"ok"}`.

Создать поездку:

```bash
curl -i -X POST localhost:8080/api/v1/trips \
  -H 'content-type: application/json' \
  -d '{
    "user_id":"5cb72c04-7650-45c9-a79b-bcdba0631e0c",
    "driver_id":"8860b315-ec86-42eb-a17c-7c163d721ff5",
    "start_point":{"latitude":59.9398,"longitude":30.3146},
    "end_point":{"latitude":59.9290,"longitude":30.3626},
    "price":1450
  }'
```

---

## Переменные окружения

Все параметры читаются из переменных окружения. Обязательная - только `DATABASE_URL`.Остальные имеют разумные значения по умолчанию, чтобы сервис запускался «из коробки» после `tripgoctl environment start`.

| Переменная | Обязательна | По умолчанию | Описание |
| `HTTP_ADDR` | нет | `:8080` | Адрес и порт HTTP-сервера |
| `LOG_LEVEL` | нет | `info` | Уровень логов: `debug`, `info`, `warn`, `error` |
| `SHUTDOWN_TIMEOUT` | нет | `10s` | Бюджет времени на graceful shutdown |
| `DATABASE_URL` | да | - | Строка подключения к PostgreSQL |
| `DATABASE_MAX_CONNS` | нет | `10` | Максимум соединений в пуле `pgxpool` |
| `DATABASE_MIN_CONNS` | нет | `2` | Минимум соединений, которые держит пул |
| `DATABASE_MAX_CONN_LIFETIME` | нет | `30m` | Время жизни одного соединения до переоткрытия |
| `DATABASE_CONNECT_TIMEOUT` | нет | `5s` | Таймаут на установку соединения и на `Ping` |
| `DATABASE_QUERY_TIMEOUT` | нет | `3s` | Таймаут на один запрос к БД |

Полный шаблон - [`.env.example`](.env.example).

---

## Команды Makefile

| Команда | Что делает |
|---|---|
| `make help` | Показать список целей |
| `make generate` | Сгенерировать `api/api.gen.go` из OpenAPI-контракта |
| `make migrate` | Накатить все миграции |
| `make migrate-down` | Откатить последнюю миграцию |
| `make migrate-status` | Показать статус миграций |
| `make build` | Собрать бинарь в `bin/trip-service` |
| `make run` | Запустить сервис, подгрузив переменные из `.env` |
| `make test` | Прогнать тесты с флагом `-race` |
| `make clean` | Удалить `bin/` |

---

## Решения

### Уровень изоляции транзакций

Используется `ReadCommitted` - уровень изоляции по умолчанию в PostgreSQL

Почему его достаточно:
- Запрет двух активных поездок у водителя держится не уровнем изоляции, а частичным уникальным индексом в БД (`trips_driver_active_uniq`). Даже при `ReadCommitted` две параллельные транзакции не смогут вставить две активные поездки одному водителю - вторая получит `unique_violation` (код `23505`).
- Защита от двойного `finish` тоже держится не изоляцией, а атомарным `UPDATE ... WHERE status = 'active'`. Один запрос обновит строку, второй увидит `RowsAffected() == 0`.
- `Serializable` дал бы более сильные гарантии, но потребовал бы логики повторных попыток при `serialization_failure`. Для нашей задачи это избыточно.

### Менеджер транзакций

Транзакции инкапсулированы в `TxManager` (пакет `internal/repository/postgres`). Интерфейс:

```go
type TxManager interface {
    Do(ctx context.Context, fn func(ctx context.Context) error) error
}
```

Как это работает:
1. `Do` открывает транзакцию и кладёт её в `context.Context` под приватным ключом (`txKey` типа `contextKey struct{}`).
2. Вызывает `fn(ctx)` - бизнес-код работает с «обогащённым» контекстом.
3. Если `fn` вернула `nil` - `COMMIT`.
4. Если `fn` вернула ошибку или паникнула - `ROLLBACK` и проброс наружу (паника пробрасывается дальше через `panic(p)`).

Бизнес-код не знает про `pgx`: в usecase пишется

```go
txManager.Do(ctx, func(ctx context.Context) error {
    tripRepo.Create(ctx, trip)
    tripRepo.AddStatusHistory(ctx, trip.ID, "", domain.TripStatusActive, "created")
    return nil
})
```

— никакой транзакции в аргументах методов репозитория.

Репозиторий достаёт исполнителя из контекста:

```go
func (r *tripRepository) getExecutor(ctx context.Context) executor {
    if tx, ok := extractTx(ctx); ok {
        return tx
    }
    return r.pool
}
```

Метод `getExecutor` возвращает либо `pgx.Tx` (если работает внутри `Do`), либо `*pgxpool.Pool` (если вне транзакции). Оба типа удовлетворяют внутреннему интерфейсу `executor` с методами `Exec`, `Query`, `QueryRow`.

Вложенный `Do` внутри другого `Do` не открывает вторую транзакцию: в начале `Do` идёт проверка `if _, ok := extractTx(ctx); ok { return fn(ctx) }` - если транзакция уже есть в контексте, используется она же. Это гарантирует атомарность всего составного сценария.

### Запрет двух активных поездок у водителя

Реализован через частичный уникальный индекс в PostgreSQL:

```sql
CREATE UNIQUE INDEX trips_driver_active_uniq
    ON trips (driver_id)
    WHERE status = 'active';
```

Индекс гарантирует: среди активных поездок `driver_id` уникален. У водителя может быть много завершённых поездок, но только одна активная одновременно.

Почему не проверка в коде: наивный подход «`SELECT` - есть ли активная, потом `INSERT`» не работает под конкурентной нагрузкой. Два параллельных запроса оба увидят «свободно», оба вставят. Проверка на уровне БД - атомарна.

Как обрабатывается в коде: при `INSERT` в `trips` вторая транзакция получает ошибку PostgreSQL с кодом `23505` (`unique_violation`). Репозиторий распознаёт её через `errors.As(err, &pgErr) && pgErr.Code == "23505"` и превращает в доменную ошибку `domain.ErrDriverBusy`. Транспортный слой маппит её в HTTP `409` с кодом `driver_busy`.

Проверка под нагрузкой: 20 параллельных `POST /api/v1/trips` на одного водителя дают ровно один `201 Created` и девятнадцать `409 Conflict`.

### Защита от двойного завершения поездки

Завершение выполняется атомарным UPDATE с условием:

```sql
UPDATE trips
   SET status = 'completed',
       finished_at = now(),
       updated_at = now()
 WHERE id = $1
   AND status = 'active';
```

Гонки нет, потому что проверка `status = 'active'` происходит внутри `UPDATE`, на уровне БД, а не отдельным `SELECT` перед ним.

Если `UPDATE` затронул `0` строк — возможны два случая:
- поездки нет → `GetByID` вернёт `ErrTripNotFound` → `404`;
- поездка есть, но она уже `completed` → возвращаем `ErrTripCompleted` → `409`.

Различение делается дополнительным чтением внутри той же транзакции.

Проверка: два параллельных `POST /api/v1/trips/{id}/finish` дают один `200 OK` и один `409 trip_completed`, `finished_at` не перезаписывается.

### Кодогенерация из OpenAPI

Весь HTTP-слой (`типы` и `chi-server`) сгенерирован из `contracts/openapi/trip-service.openapi.yaml` через `oapi-codegen`. Сгенерированный файл - [`api/api.gen.go`](api/api.gen.go), коммитится в репозиторий.

Руками сгенерированный код не правится: любое изменение контракта → `make generate` → файл перезаписывается. За счёт этого код и контракт не могут разъехаться.

Для ошибок парсинга path-параметров (например, `tripId` не UUID) `oapi-codegen` по умолчанию возвращает `text/plain`. Мы передаём кастомный `ErrorHandlerFunc` через `api.ChiServerOptions` - он возвращает `application/problem+json` с кодом `invalid_request`. Сгенерированный код при этом не правится.

### Graceful shutdown

Сервис ловит `SIGINT` и `SIGTERM` через `signal.NotifyContext`. По сигналу:
1. Перестаём принимать новые соединения.
2. Ждём завершения активных запросов (`http.Server.Shutdown`).
3. Закрываем пул соединений с БД (`defer pool.Close()`).

Бюджет времени на шаг 2 - `SHUTDOWN_TIMEOUT` (по умолчанию `10s`). По истечении - сервис завершается принудительно.

### Обработка ошибок

Ошибки отдаются по RFC 9457 в `application/problem+json`. Схема `Problem` - в контракте.

Таблица маппинга доменных ошибок в HTTP:

| Доменная ошибка | HTTP | `code` |
|---|---|---|
| `ErrTripNotFound` | `404` | `trip_not_found` |
| `ErrDriverBusy` | `409` | `driver_busy` |
| `ErrTripCompleted` | `409` | `trip_completed` |
| ошибка валидации в хендлере | `400` | `invalid_request` |
| любая другая | `500` | `internal_error` |

Маппинг собран в одном месте - метод `Handlers.writeError`. `Problem` всегда содержит поля `type`, `title`, `status`, `code`, `detail`, `instance`. Внутренние детали (SQL, стектрейсы) в ответ клиенту не попадают - только в лог.

### Логирование

`log/slog`, JSON-handler, вывод в stdout. Уровень - из `LOG_LEVEL`.

Логи структурированные: каждая запись содержит `time`, `level`, `msg` и дополнительные поля (`addr`, `error`, `trip_id` и т. п.). Это позволяет фильтровать и агрегировать логи во внешних системах (Loki, ELK и др.). Персональные данные в логи не пишутся.

---

## Структура проекта

```
cmd/trip-service/          точка входа (main)
internal/
  config/                  чтение конфигурации из переменных окружения
  domain/                  доменные типы и ошибки (Trip, TripStatus, ErrXxx)
  usecase/                 бизнес-логика (TripUsecase)
  transport/http/          HTTP-слой: хендлеры, роутер, сервер
  repository/postgres/     работа с БД: пул, репозиторий, менеджер транзакций
api/                       сгенерированный код (api.gen.go)
migrations/                SQL-миграции goose
contracts/                 контракты курса (OpenAPI, schema.md)
config/                    примеры env-файлов, конфиг линтера
deploy/                    Dockerfile (будет в ЛР1*)
Makefile                   команды сборки/запуска/миграций
.env.example               шаблон переменных окружения
environment.toml           описание локального окружения для tripgoctl
```

---

## Что сделано в ЛР1

- Инициализирован Go-модуль, настроена структура репозитория по `conventions.md` §8.
- Конфигурация из переменных окружения (`internal/config`), с валидацией обязательных полей на старте.
- HTTP-сервер на `chi` с таймаутами и graceful shutdown.
- Ручки `/health` и `/ready` (с проверкой доступности БД).
- Кодогенерация типов и серверного интерфейса из OpenAPI (`api/api.gen.go`).
- Миграции `trips` и `trip_status_history` (goose, up + down).
- Ограничение против двух активных поездок - partial unique index.
- Подключение к PostgreSQL через `pgxpool` с `Ping` на старте.
- Репозиторий на `pgx` + `squirrel` с обработкой `23505` и `pgx.ErrNoRows`.
- Менеджер транзакций `TxManager` через `context.Context` (вложенный `Do` переиспользует транзакцию).
- Три бизнес-ручки: `createTrip`, `getTrip`, `finishTrip` - с кодами `200/201/400/404/409/500`.
- Ошибки в формате `application/problem+json` (RFC 9457) с машиночитаемым `code`.
- Логи в JSON (`log/slog`) в stdout, уровень из `LOG_LEVEL`.
- Проверена конкурентность: 20 параллельных `POST /trips` на одного водителя → 1×`201` + 19×`409`.

---

## Что не сделано

- **Идемпотентность `POST /api/v1/trips` по заголовку `Idempotency-Key`** - задание со звёздочкой ЛР1, не реализовано.
- **Dockerfile** - задание со звёздочкой ЛР1, не реализовано.