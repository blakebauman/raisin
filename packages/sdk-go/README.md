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
