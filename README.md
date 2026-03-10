# Subs Manager Service

Сервис для управления подписками.

# Описание
- CRUDL для подписки
- Отдельная ручка для посчета суммарной стоимости подписок с фильтром.
- Rate Limiter реализован через Token Bucket (по дефолту стоит 10 токенов, восстановление по 1-ому за 20сек)

## Технологии
- **Go 1.26.1**
- **PostgreSQL 16** 
- **Docker** и **Docker Compose** 

## Установка и запуск

### Запуск через Docker Compose

1. Запустите сервис `make docker-up` или `docker-compose up -d --build`
2. Остановка сервиса `make docker-down` или `docker-compose down -v`

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
│   ├── rateLimiting/   # Rate Limiter
│   ├── repository/     # Репозиторий
│   ├── service/        # Бизнес-логика
│   ├── logger/         # Инициализация логгера
│   ├── utils/          # Вспомогательные функции
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
