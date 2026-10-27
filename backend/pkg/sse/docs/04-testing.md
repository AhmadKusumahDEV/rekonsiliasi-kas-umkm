# Testing

```bash
go test ./pkg/sse -count=1
go test -race ./pkg/sse -count=1
golangci-lint run ./pkg/sse/...
go test ./pkg/sse -run '^$' -bench 'BenchmarkHubBroadcast1000' -benchmem
```

The suite uses `httptest` and real HTTP streams over loopback. It needs no
database or broker. If the sandbox blocks loopback TCP, run the tests with
network permission; the error is environmental, not a package failure.

`TestMain` runs `goleak`. CI runs the race detector and a short fuzz pass for
the event encoder. The 1000-client benchmark is synthetic fanout, not a network
capacity result.

Coverage includes:

- authenticated stream setup and 401/500/503 mapping,
- cross-origin rejection,
- per-user and per-room delivery to multiple streams,
- `JoinUser`/`LeaveUser`/`CloseRoom`,
- `Last-Event-ID` replay,
- token expiry 4001 and connection age 4002,
- close-event retry jitter,
- replay failure/truncation `resync`,
- replay event-size failure `resync`,
- replay requires application event IDs when configured,
- `OnReject` reasons and `Retry-After` on 429/503,
- HTTP middleware ordering and `OnPublish` observations,
- per-type event data limits,
- `Client.CloseInfo` and symmetric `OnDisconnect` behavior,
- jittered retry for 1012 and connection age,
- graceful drain of the last queued event,
- slow client 1013 end-to-end through the stream writer,
- slow-client `OnDrop` and queue overflow,
- client context and disconnect cleanup,
- Bus publish/apply and Presence add/remove,
- Presence heartbeat refresh,
- CORS preflight and Last-Event-ID length rejection,
- real `http.Server` shutdown through `RegisterOnShutdown`,
- `ResponseWriter` without flush support,
- shutdown code 1012,
- configuration validation.

Add a real integration test in the host project for the project's auth, replay
store and broker adapter. Package tests cannot validate deployment-specific
durability or load limits.
