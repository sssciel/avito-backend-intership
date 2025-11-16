# avito-backend-intership
## Описание

Микросервис для автоматического назначения ревьюеров на Pull Requestы с управлением командами и участниками.

## Гайд по запуску

### Запуск через Docker

```bash
# Запуск всего приложения (база данных + сервис)
docker-compose up -d

# Приложение будет доступно на порту 8080
curl http://localhost:8080/health
```

### Без Docker

```bash
# Установите зависимости
go mod download

# Настройте переменные окружения в cfg/.env
cp cfg/.env.example cfg/.env
# Отредактируйте cfg/.env под свои параметры

# Запустите PostgreSQL (если не используете Docker)
# Примените миграции вручную
# go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest
# migrate -source file://migrations -database "postgres://name:pass@localhost/dbname" up 

# Запустите приложение
make run
# или
go run cmd/main.go
```

## Тестирование
### Юнит-тесты

Тесты для моделей, сервисов и storage слоев с использованием моков:

```bash
# Запуск только юнит-тестов
make test-unit
```

### Интеграционные тесты

E2E тесты, работающие с реальной базой данных:

```bash
# Запуск интеграционных тестов (автоматически поднимет тестовую БД)
make test-integration
```

### Все тесты

```bash
# Запуск всех тестов (юнит + интеграционные)
make test
```

## API Endpoints

Документация API находится в `docs/openapi.yml`.

### Основные эндпоинты:

- `POST /api/v1/team/add` - Создание команды с участниками
- `GET /api/v1/team/get?team_name=<name>` - Получение команды
- `POST /api/v1/users/setIsActive` - Установка статуса активности пользователя
- `GET /api/v1/users/getReview?user_id=<id>` - Получение PR'ов пользователя
- `POST /api/v1/pullRequest/create` - Создание PR с автоназначением ревьюеров
- `POST /api/v1/pullRequest/merge` - Мерж PR (идемпотентная операция)
- `POST /api/v1/pullRequest/reassign` - Переназначение ревьювера
- `GET /health` - Health check

## Структура проекта

```
.
├── cmd/
│   └── main.go                 # Точка входа приложения
├── internals/
│   ├── pullrequests/          # Сервис PR
│   │   ├── pullrequests.go
│   │   └── pullrequests_test.go
│   ├── teams/                 # Сервис команд
│   │   ├── service.go
│   │   └── service_test.go
│   ├── users/                 # Сервис пользователей
│   │   ├── users.go
│   │   └── users_test.go
│   └── storage/               # Слой хранения данных
│       ├── storage.go         # Интерфейсы
│       ├── models/            # Модели данных
│       ├── pgsql/             # PostgreSQL реализация
│       └── mocks/             # Моки для тестов
├── tests/
│   └── integration_test.go    # Интеграционные тесты
├── migrations/                # SQL миграции
├── docs/
│   └── openapi.yml           # OpenAPI спецификация
├── cfg/
│   ├── .env                  # Конфигурация (production)
│   ├── .env.example          # Пример конфигурации
│   └── .env.test             # Конфигурация для тестов
├── pkg/
│   └── config/               # Утилиты конфигурации
├── docker-compose.yaml       # Docker Compose для приложения
├── docker-compose.test.yml   # Docker Compose для тестов
├── Makefile                  # Команды сборки и тестирования
└── README.md
```

## Особенности реализации

### Назначение ревьюеров

- При создании PR автоматически назначается до 2 активных ревьюеров из команды автора
- Автор PR исключается из списка кандидатов
- Учитывается только активные пользователи (`is_active = true`)

### Переназначение ревьюверов

- Заменяет одного ревьювера на случайного активного участника из команды заменяемого
- Невозможно после мерджа PR
- Возвращает ошибку `NO_CANDIDATE`, если нет доступных кандидатов

### Идемпотентность

- Операция merge PR идемпотентна - повторный вызов возвращает актуальное состояние без ошибки

### Тестирование

- **Юнит-тесты**: Используют моки для изоляции тестируемых компонентов
- **Интеграционные тесты**: Проверяют полный flow с реальной БД
- **Покрытие**: Включает тесты основных сценариев и граничных случаев

## Переменные окружения

```bash
DB_HOST=localhost          # Хост базы данных
DB_PORT=5432              # Порт базы данных
DB_USER=admin             # Пользователь БД
DB_PASSWORD=admin         # Пароль БД
DB_NAME=avito             # Название БД
SERVICE_PORT=8080         # Порт сервиса
SERVICE_API_TOKEN=token   # API токен (опционально)
```
