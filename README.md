# go-musthave-metrics-tpl

Шаблон репозитория для трека «Сервер сбора метрик и алертинга».

## Начало работы

1. Склонируйте репозиторий в любую подходящую директорию на вашем компьютере.
2. В корне репозитория выполните команду `go mod init <name>` (где `<name>` — адрес вашего репозитория на GitHub без префикса `https://`) для создания модуля.

## Обновление шаблона

Чтобы иметь возможность получать обновления автотестов и других частей шаблона, выполните команду:

```
git remote add -m v2 template https://github.com/Yandex-Practicum/go-musthave-metrics-tpl.git
```

Для обновления кода автотестов выполните команду:

```
git fetch template && git checkout template/v2 .github
```

Затем добавьте полученные изменения в свой репозиторий.

## Запуск автотестов

Для успешного запуска автотестов называйте ветки `iter<number>`, где `<number>` — порядковый номер инкремента. Например, в ветке с названием `iter4` запустятся автотесты для инкрементов с первого по четвёртый.

При мёрже ветки с инкрементом в основную ветку `main` будут запускаться все автотесты.

Подробнее про локальный и автоматический запуск читайте в [README автотестов](https://github.com/Yandex-Practicum/go-autotests).

## Информация о сборке

При старте `server` и `agent` печатают в stdout:

```
Build version: ...
Build date: ...
Build commit: ...
```

По умолчанию значения равны `N/A`. Их можно задать при компиляции через `-ldflags`:

```bash
go build -ldflags="-X main.buildVersion=1.0.0 -X main.buildDate=$(date -u +%Y-%m-%dT%H:%M:%SZ) -X main.buildCommit=$(git rev-parse --short HEAD)" -o cmd/server/server ./cmd/server

go build -ldflags="-X main.buildVersion=1.0.0 -X main.buildDate=$(date -u +%Y-%m-%dT%H:%M:%SZ) -X main.buildCommit=$(git rev-parse --short HEAD)" -o cmd/agent/agent ./cmd/agent
```

Или через Makefile (подставит VERSION/DATE/COMMIT автоматически):

```bash
make build-server
make build-agent
# либо явно:
make build-server VERSION=1.0.0 COMMIT=abc1234
```

В Docker:

```bash
docker build \
  --build-arg VERSION=1.0.0 \
  --build-arg DATE=$(date -u +%Y-%m-%dT%H:%M:%SZ) \
  --build-arg COMMIT=$(git rev-parse --short HEAD) \
  --target server \
  -t metrics-server .
```

## Структура проекта

Приведённая в этом репозитории структура проекта является рекомендуемой, но не обязательной.

Это лишь пример организации кода, который поможет вам в реализации сервиса.

При необходимости можно вносить изменения в структуру проекта, использовать любые библиотеки и предпочитаемые структурные паттерны организации кода приложения, например:
- **DDD** (Domain-Driven Design)
- **Clean Architecture**
- **Hexagonal Architecture**
- **Layered Architecture**
