# junge-checkers

[![junge-checkers](https://github.com/skipper-ad/junge-checkers/actions/workflows/go.yml/badge.svg)](https://github.com/skipper-ad/junge-checkers/actions/workflows/go.yml)

Go-библиотека для написания checker-ов под Skipper AD и Forcad.  
Вдохновлена [go-checklib](https://github.com/pomo-mondreganto/go-checklib)

## Установка

```bash
go get github.com/skipper-ad/junge-checkers@latest
```

Для локальной разработки:

```bash
git clone https://github.com/skipper-ad/junge-checkers.git
cd junge-checkers
go test ./...
```

## Минимальный checker

```go
package main

import junge "github.com/skipper-ad/junge-checkers"

func main() {
	junge.Main(junge.Handler{
		Config: junge.CheckerInfo{
			Vulns:      1,
			Timeout:    10,
			AttackData: true,
			Puts:       1,
			Gets:       1,
		},
		CheckFunc: func(c *junge.C, host string) {
			c.OK("OK")
		},
		PutFunc: func(c *junge.C, req junge.PutRequest) {
			c.OK(req.FlagID)
		},
		GetFunc: func(c *junge.C, req junge.GetRequest) {
			c.OK("OK")
		},
	})
}
```

## CLI-контракт

Skipper вызывает checker так:

```text
checker info
checker check <host>
checker put <host> <flag_id> <flag> <vuln>
checker get <host> <flag_id> <flag> <vuln>
```

Где:

- `host` - IP или hostname команды;
- `flag_id` - данные, которые `put` вернул в public stdout и которые потом получит `get`;
- `flag` - сам флаг;
- `vuln` - номер vuln/place.

`junge.Run` строго валидирует количество аргументов. Лишние или отсутствующие аргументы приводят к `CHECK FAILED`, потому что это ошибка запуска checker-а, а не сервиса.

## Verdict-ы

| Verdict | Exit code | Когда использовать |
| --- | ---: | --- |
| `OK` | `101` | Всё хорошо |
| `CORRUPT` | `102` | Сервис жив, но сохранённый флаг потерян или изменён |
| `MUMBLE` | `103` | Сервис отвечает, но нарушает ожидаемый протокол или формат |
| `DOWN` | `104` | Сервис недоступен или отвечает серверной ошибкой |
| `CHECK FAILED` | `110` | Ошибка checker-а: неверные аргументы, panic, баг в логике |

Методы завершения:

```go
c.OK("OK")
c.Corruptf("Flag was corrupted", "flag_id=%s not found", req.FlagID)
c.Mumble("Bad response", "invalid JSON from /api/profile")
c.Down("Service is down", "connect timeout")
c.CheckFailed("Checker failed", "unexpected local invariant")
```

Если private message не передан, он будет равен public message.

Для подробной диагностики можно добавлять структурированные детали. Они попадут в private output:

```go
c.Detail("flag_id", req.FlagID)
c.Detail("vuln", req.Vuln)
c.Corrupt("Flag was corrupted")
```

Private output:

```text
Flag was corrupted
flag_id=note-42
vuln=1
```

## Handler API

`Handler` - основной рекомендуемый способ писать checker-ы на `junge`.

Внешний контракт checker-а процедурный: Skipper запускает один и тот же бинарь с action-ами `info`, `check`, `put`, `get`. Внутри Go-кода это удобно представить как набор функций:

- `CheckFunc` проверяет доступность и базовую корректность сервиса;
- `PutFunc` кладёт флаг и возвращает `flag_id` через `c.OK(flagID)`;
- `GetFunc` получает `flag_id`, читает сохранённый флаг и сравнивает его с ожидаемым;
- `Config` описывает `info`.

Поэтому `Handler` сделан как маленький adapter между CLI-контрактом Skipper и обычными Go-функциями. Разработчику не нужно писать `switch os.Args[1]`, вручную печатать stdout/stderr или помнить exit codes. При этом все action-ы остаются явно разделены, а сигнатуры не дают случайно перепутать `put` и `get` аргументы.

```go
junge.Main(junge.Handler{
	Config: junge.CheckerInfo{Vulns: 1, Timeout: 10, Puts: 1, Gets: 1},
	CheckFunc: func(c *junge.C, host string) {
		c.OK("OK")
	},
	PutFunc: func(c *junge.C, req junge.PutRequest) {
		c.OK(req.FlagID)
	},
	GetFunc: func(c *junge.C, req junge.GetRequest) {
		c.OK("OK")
	},
})
```

`junge.Main` вызывает `os.Exit`. Для unit-тестов используйте `junge.RunWithArgs`.

Если checker становится большим, `Handler` всё равно можно оставить как точку сборки, а бизнес-логику вынести в обычные функции и пакеты внутри проекта checker-а.

## Info

```go
junge.CheckerInfo{
	Vulns:      2,     // количество уязвимостей/places
	Timeout:    10,    // timeout action в секундах
	AttackData: true,  // публиковать ли flag_id как attack data
	Puts:       1,     // сколько PUT запускать за раунд
	Gets:       1,     // сколько GET запускать за раунд
}
```

`info` печатает JSON:

```json
{"vulns":1,"timeout":10,"attack_data":true,"puts":1,"gets":1}
```

## Assertions

Пакет `require` завершает checker, если условие не выполнено. По умолчанию ошибка assertion-а даёт `MUMBLE`; статус можно переопределить.

```go
import (
	"github.com/skipper-ad/junge-checkers/require"
	o "github.com/skipper-ad/junge-checkers/require/options"
)

require.NoError(c, err, "Service is down", o.Down())
require.Equal(c, expected, actual, "Flag was corrupted", o.Corrupt())
require.NotEqual(c, "", id, "Could not save flag", o.Corrupt())
require.Contains(c, body, flag, "Flag was corrupted", o.Corrupt())
require.Greater(c, len(items), 0, "Empty list")
```

Опции:

```go
o.Corrupt()
o.Mumble()
o.Down()
o.CheckFailed()
o.Private("private details")
o.Privatef("flag_id=%s: %v", flagID, err)
o.Status(junge.StatusCorrupt)
```

## Генераторы

```go
import "github.com/skipper-ad/junge-checkers/gen"

login := gen.Username()
password := gen.Password(24)
token := gen.String(32)
hex := gen.StringAlphabet(16, gen.HexAlphabet)
agent := gen.UserAgent()
title := gen.Sentence()
text := gen.Paragraph()
pick := gen.Sample([]string{"red", "green", "blue"})
number := gen.RandInt(1, 10) // inclusive
raw := gen.Bytes(c, 16)
```

Генераторы используют `crypto/rand` и не требуют seed-а.

## HTTP helpers

Пакет `httpx` закрывает типовой HTTP checker flow: base URL, timeout, random user agent, JSON decoding и перевод HTTP-ошибок в корректные verdict-ы.

```go
api := httpx.NewClient(c, "http://"+host+":8080")

resp := api.Get("/health")
httpx.ExpectStatus(c, resp, 200, "Service is unhealthy")

resp = api.PostJSON("/api/notes", map[string]string{"text": flag})
var created struct {
	ID string `json:"id"`
}
httpx.JSON(c, resp, &created, "Could not save flag", o.Corrupt())
```

Можно использовать typed service wrapper, чтобы не создавать client вручную в каждом action:

```go
junge.Main(httpx.Service{
	Port: 8080,
	ClientOptions: []httpx.Option{
		httpx.WithCookieJar(),
		httpx.WithRetries(2, 100*time.Millisecond),
	},
	CheckFunc: func(c *junge.C, api *httpx.Client) {
		api.ExpectStatus(api.Get("/health"), 200, "Service is unhealthy")
		c.OK("OK")
	},
	PutFunc: func(c *junge.C, api *httpx.Client, req junge.PutRequest) {
		// save req.Flag
	},
	GetFunc: func(c *junge.C, api *httpx.Client, req junge.GetRequest) {
		// read req.FlagID and compare with req.Flag
	},
})
```

Правила `httpx`:

- network error -> `DOWN`;
- HTTP `5xx` -> `DOWN`;
- HTTP `4xx` -> по умолчанию `MUMBLE`, но можно передать `o.Corrupt()`;
- invalid JSON/text -> по умолчанию `MUMBLE`.

Дополнительные возможности `httpx`:

- cookie session: `httpx.WithCookieJar()`;
- retries для idempotent requests: `httpx.WithRetries(3, 100*time.Millisecond)`;
- auth helpers: `httpx.WithBearerToken(token)`, `httpx.WithBasicAuth(login, password)`;
- forms: `api.PostForm("/login", url.Values{"login": []string{"alice"}})`;
- multipart: `api.PostMultipart(...)`;
- snippets response body в diagnostics: `httpx.WithErrorBodySnippet(512)`.

## Пример

Полный пример: [examples/simple_http_checker/main.go](examples/simple_http_checker/main.go).

Он ожидает сервис заметок:

- `GET /health` -> `200`;
- `POST /api/notes {"text":"FLAG"}` -> `{"id":"..."}`;
- `GET /api/notes/{id}` -> `{"id":"...","text":"FLAG"}`.

Дополнительные примеры:

- [examples/service_wrapper_checker](examples/service_wrapper_checker) - checker на `httpx.Service`;
- [examples/tcp_checker](examples/tcp_checker) - checker для line-based TCP protocol.

## Тестирование checker-ов

Пакет `checkertest` упрощает unit-тесты checker-ов:

```go
res := checkertest.Check(t, checker, "127.0.0.1")
res.RequireOK(t)
res.RequirePublic(t, "OK")

put := checkertest.Put(t, checker, "127.0.0.1", "id", "FLAG", 1)
put.RequireOK(t)
```

Есть helpers для `Info`, `Check`, `Put`, `Get`, проверки статусов, public/private output и parsing `CheckerInfo`.

## Структура checker-репозитория

Рекомендуемая структура - отдельный Go module на каждый сервис. То есть `go.mod` и `go.sum` лежат внутри папки конкретного checker-а:

```text
checkers/
  notes/
    go.mod
    go.sum
    checker.go
    client.go
    models.go
  shop/
    go.mod
    go.sum
    cmd/
      checker/
        main.go
    internal/
      api/
        client.go
```
