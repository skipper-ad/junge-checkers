# service_wrapper_checker

Example checker built with `httpx.Service`.

It removes repetitive client setup from each action:

```go
junge.Main(httpx.Service{
	Port: 8080,
	CheckFunc: check,
	PutFunc:   put,
	GetFunc:   get,
})
```

The external Skipper contract is unchanged. The wrapper only creates a typed `*httpx.Client` for every action.

