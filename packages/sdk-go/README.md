# Raisin Go SDK

Module: `github.com/blakebauman/raisin-go`

```go
import "github.com/blakebauman/raisin-go"

client := raisin.NewClient("ra_…")
client.BaseURL = "http://localhost:18080"
```

From another Go module in this monorepo (or a sibling checkout):

```go
// go.mod
require github.com/blakebauman/raisin-go v0.0.0
replace github.com/blakebauman/raisin-go => ./packages/sdk-go
```

Webhook verification:

```go
ok := raisin.VerifyWebhookSignature(secret, header, body, 0)
```

Live event stream (SSE) — cancel the context to disconnect; pass `LastEventID` on reconnect:

```go
ctx, cancel := context.WithCancel(context.Background())
defer cancel()
ch, errs, err := client.Events.Stream(ctx, &raisin.StreamOpts{
	Types: []string{"email.opened", "email.clicked", "email.delivered"},
})
if err != nil { log.Fatal(err) }
for {
	select {
	case ev, ok := <-ch:
		if !ok { return }
		fmt.Println(ev.Type, string(ev.Data))
	case err := <-errs:
		if err != nil { log.Println(err) }
		return
	}
}
```
