# Wishlist API

REST API для создания вишлистов к праздникам и событиям. Регистрируешься, создаёшь вишлист, добавляешь подарки, делишься ссылкой — друзья видят список и могут забронировать подарок.

## Стек

- Go 1.21+
- PostgreSQL
- Docker Compose
- JWT авторизация
- Миграции через goose

## Запуск

```bash
docker-compose up --build
```

Сервер стартует на `http://localhost:8080`. Миграции применяются автоматически.

## API

Все эндпоинты начинаются с `/api/v1`. Ответы в JSON.

### Auth

```bash
# Регистрация
curl -s -X POST http://localhost:8080/api/v1/auth/register \
  -H "Content-Type: application/json" \
  -d '{"email": "user@example.com", "password": "secret123"}'

# Логин
curl -s -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email": "user@example.com", "password": "secret123"}'
```

В ответе приходит JWT-токен. Дальше передаёшь его в заголовке `Authorization: Bearer <token>`.

### Вишлисты

```bash
# Создать
curl -s -X POST http://localhost:8080/api/v1/wishlists \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"title": "День рождения", "description": "Хочу", "event_date": "2026-06-15"}'

# Список своих
curl -s http://localhost:8080/api/v1/wishlists \
  -H "Authorization: Bearer $TOKEN"

# Получить по id
curl -s http://localhost:8080/api/v1/wishlists/$ID \
  -H "Authorization: Bearer $TOKEN"

# Удалить
curl -s -X DELETE http://localhost:8080/api/v1/wishlists/$ID \
  -H "Authorization: Bearer $TOKEN"
```

### Подарки

```bash
# Добавить подарок в вишлист
curl -s -X POST http://localhost:8080/api/v1/wishlists/$ID/items \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"title": "Клавиатура", "url": "https://example.com/kb", "priority": 5}'
```

Приоритет от 1 до 5, по умолчанию 3.

### Публичный доступ

При создании вишлиста генерируется `share_token`. По нему можно смотреть вишлист и бронировать подарки без авторизации:

```bash
# Посмотреть вишлист
curl -s http://localhost:8080/api/v1/shared/$SHARE_TOKEN

# Забронировать подарок
curl -s -X POST http://localhost:8080/api/v1/shared/$SHARE_TOKEN/items/$ITEM_ID/reserve
```

## Тесты

```bash
go test ./...
```
