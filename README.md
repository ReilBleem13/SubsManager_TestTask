# Subs Manager Service

Сервис для управления подписками.

# Описание
- CRUDL для подписки
- Отдельная ручка для посчета суммарной стоимости подписок с фильтром.

## Технологии
- **Go 1.26.1**
- **PostgreSQL 16** 
- **Docker** и **Docker Compose** 

## Установка и запуск

### Запуск через Docker Compose

1. Создайте .env в корне проекта со следующим содержимым (настройте под себя):
    ```env
    APP_PORT=8080
    APP_LOG_LEVEL=debug

    DATABASE_HOST=db
    DATABASE_PORT=5432
    DATABASE_USER=colorvax
    DATABASE_PASSWORD=colorvax
    DATABASE_DBNAME=colorvax
    DATABASE_SSLMODE=disable

2. Запустите сервис `make docker-up` или `docker-compose up -d --build`
3. Остановка сервиса `make docker-down` или `docker-compose down -v`

## Миграции 

Миграции базы данных выполняются автоматически при запуске приложения с использованием [goose](https://github.com/pressly/goose).
Миграции находятся в директории `internal/infra/database/migrations/`

## Структура проекта

```
.
├── cmd/
│   └── app/            # Основа
├── docs/               # Swagger
├── internal/
│   ├── config/         # Конфигурация
│   ├── domain/         # Доменные модели
│   ├── transport/      # HTTP handlers
│   ├── repository/     # Репозиторий
│   ├── service/        # Бизнес-логика
│   ├── logger/         # Инициализация логгера
│   └── infra/          # Инфраструктура (клиент БД)
├── migrations/         # Миграции базы данных
├── docker-compose.yaml
├── Dockerfile
├── Makefile
└── README.md
```

## Разработка

### Запуск тестов

1. Запуск всех тестов `make test` 