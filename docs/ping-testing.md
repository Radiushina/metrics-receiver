# Как тестировать запрос `ping`

1. Запустите сервер с настроенным подключением к БД: переменная **`DATABASE_DSN`** или флаг **`-d`** со строкой PostgreSQL (DSN).

Пример через терминал (пользователь `developer`, пароль `my_pass`, база `metrics` — как в docker-compose.yml):

```bash
export DATABASE_DSN='postgres://developer:my_pass@localhost:5432/metrics?sslmode=disable'
go run ./cmd/server
```

2. В другом терминале выполните:

```bash
curl -i http://localhost:8080/ping
```

Если сервер слушает другой адрес или порт, замените URL (по умолчанию порт **8080**, флаг **`-a`**).

3. Смотрите код ответа: при успешной проверке БД — **200**, при ошибке — **500**.
