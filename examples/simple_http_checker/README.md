# simple_http_checker

Пример простого HTTP checker-а для сервиса заметок.

Ожидаемый API сервиса:

```text
GET  /health                  -> 200
POST /api/notes {"text": "..."} -> {"id":"..."}
GET  /api/notes/{id}          -> {"id":"...","text":"..."}
```

Команды:

```bash
go run . info
go run . check 127.0.0.1
go run . put 127.0.0.1 initial-id FLAG 1
go run . get 127.0.0.1 saved-id FLAG 1
```

`check` проверяет `/health`.

`put` кладёт флаг в `/api/notes` и возвращает ID созданной записи. Этот ID checksystem сохранит как `flag_id`.

`get` читает `/api/notes/{flag_id}` и сравнивает `text` с исходным флагом.

