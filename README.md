# users-notifier

HTTP-сервис для отправки сообщений пользователям через Telegram-бота. Принимает запросы с идентификатором чата и текстом, проксирует их в Telegram Bot API и экспортирует метрики в формате Prometheus.

## Что делает сервис

- Принимает `POST /send` с JSON-телом `{"chat_id": ..., "text": "..."}` и отправляет сообщение в указанный Telegram-чат
- Экспортирует метрики: количество успешных и неудачных отправок, гистограмму времени ответа Bot API
- Отвечает на `GET /healthz` для проверки работоспособности

## Конфигурация

Все настройки задаются в `config.yaml`:

```yaml
telegram:
  bot_token: ""   # токен бота от @BotFather

server:
  port: 8080      # порт API

logging:
  level: "info"   # debug / info / warn / error
  file_path: "/var/log/users-notifier/app.log"

metrics:
  port: 9090      # порт Prometheus-метрик
```

Токен бота можно получить у [@BotFather](https://t.me/BotFather) командой `/newbot`.

## API

### POST /send

Отправляет сообщение в Telegram-чат.

**Тело запроса:**
```json
{
  "chat_id": 123456789,
  "text": "Текст сообщения"
}
```

**Ответы:**
- `200 OK` — сообщение успешно отправлено
- `400 Bad Request` — невалидный JSON, отсутствует `chat_id` или `text`
- `502 Bad Gateway` — ошибка при обращении к Telegram Bot API

**Пример:**
```bash
curl -X POST http://localhost:8080/send \
  -H 'Content-Type: application/json' \
  -d '{"chat_id": 123456789, "text": "Привет!"}'
```

### GET /healthz

```bash
curl http://localhost:9090/healthz
# ok
```

### GET /metrics

Prometheus-метрики на порту `9090`:

```bash
curl http://localhost:9090/metrics
```

| Метрика | Тип | Описание |
|---|---|---|
| `notifier_messages_sent_total` | Counter | Успешно отправленных сообщений |
| `notifier_messages_failed_total` | Counter | Неудачных попыток отправки |
| `notifier_send_duration_seconds` | Histogram | Время ответа Telegram Bot API |

---

## Запуск в Docker

Сервис запускается в Docker-сети `realty-net` (общей с остальными сервисами).

### Сборка и запуск

```bash
bash server_setup.sh
```

Скрипт собирает образ и запускает контейнер `users-notifier`. Логи пишутся в `/tmp/users-notifier-logs/` на хосте.

```
# Логи:
docker logs -f users-notifier

# Остановить:
docker stop users-notifier
```

### Порты

| Порт на хосте | Порт в контейнере | Назначение |
|---|---|---|
| `8080` | `8080` | API (`/send`) |
| `9091` | `9090` | Метрики и healthz |