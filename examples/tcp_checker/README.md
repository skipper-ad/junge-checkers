# tcp_checker

Example checker for a simple line-based TCP service.

Expected protocol:

```text
PING              -> PONG
PUT <id> <flag>   -> OK
GET <id>          -> <flag>
```

The example uses `net.Dialer.DialContext` with `*junge.C`, so checker timeouts cancel network calls correctly.

