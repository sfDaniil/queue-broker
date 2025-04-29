# Queue‑Broker

[![CI](https://github.com/sfdaniil/queue-broker/actions/workflows/ci.yml/badge.svg)](https://github.com/sfdaniil/queue-broker/actions)

Минимальный HTTP‑брокер сообщений, написанный на **Go** — без сторонних зависимостей, один бинарник, готовый к Docker.

## Возможности

* **PUT /queue/{name}** – помещает JSON `{"message": "..."}` в очередь  
* **GET /queue/{name}?timeout=N** – извлекает сообщения FIFO, поддерживает long‑polling  
* Настраивается порт, таймаут по умолчанию, лимит очередей и сообщений  
* In‑memory‑хранилище с защитой от переполнения  
* Graceful‑shutdown  
* Только стандартная библиотека Go  
* Unit‑тесты, Makefile для lint/coverage/Docker‑сборки

## Быстрый старт

```bash
make build            # локальный бинарник
./queue-broker -port 8080 -timeout 30s
```

Или в Docker (образ <10 МБ):

```bash
make docker-run PORT=8080
```

## API

### Добавить сообщение в очередь

```bash
curl -X PUT http://localhost:8080/queue/pets \
     -H "Content-Type: application/json" \
     -d '{"message":"Garfield"}'
# → 200 OK
```

Ошибки

| Код | Причина                              |
|-----|--------------------------------------|
| 400 | неверный JSON / пустое поле          |
| 429 | превышен лимит очередей/сообщений    |

### Получить сообщение из очереди

```bash
curl http://localhost:8080/queue/pets?timeout=10
# → {"message":"Garfield"}
```

Ошибки

| Код | Причина                                      |
|-----|----------------------------------------------|
| 404 | нет сообщения до истечения таймаута          |

## Флаги командной строки

| Флаг              | Значение по умолчанию | Описание                                           |
|-------------------|-----------------------|----------------------------------------------------|
| `-port`           | `8080`                | HTTP‑порт                                          |
| `-timeout`        | `30s`                 | Таймаут GET по умолчанию (можно переопределить)    |
| `-max-queues`     | `0`                   | Лимит количества очередей (`0` — без ограничений)  |
| `-max-messages`   | `0`                   | Лимит сообщений **на очередь** (`0` — без ограничений) |

## Структура проекта

```Text
.
├── cmd/server        # точка входа (флаги, HTTP‑mux)
└── internal
    ├── queue         # логика FIFO-очереди
    └── broker        # брокер, HTTP‑обработчик, ошибки
```

## Разработка

```bash
make test            # юнит‑тесты + покрытие
make lint            # golangci‑lint
make cover-html      # HTML‑отчёт покрытия
```

### Docker

```bash
make docker-build            # сборка образа
make docker-run PORT=8080    # запуск контейнера
```
