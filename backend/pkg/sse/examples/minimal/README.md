# Minimal SSE example

The smallest end-to-end wiring: `New`, `Start`, mount `/events`,
`PublishEventToUser`, and `Shutdown`. It has no rooms, replay, bus or HTML.

```bash
go run ./pkg/sse/examples/minimal
```

Use another port with `ADDR=:8091` on Linux or macOS, or
`$env:ADDR=":8091"` in PowerShell.

Terminal 1, keep the stream open:

```bash
curl -N -H "X-User: ana" http://localhost:8090/events
```

Terminal 2, send a notice to that user:

```bash
curl -X POST "http://localhost:8090/notify?user=ana"
```

Terminal 2 answers 204 and Terminal 1 receives `event: notice`. Stop the server
with Ctrl+C; it drains the stream before shutting down HTTP.

| Call | Why |
| --- | --- |
| `sse.New` | Validate config and build the manager |
| `manager.Start` | Mark ready and subscribe to an optional bus |
| `mux.Handle("/events", manager)` | Open the stream |
| `manager.PublishEventToUser` | Send to every device of one user |
| `manager.Shutdown` | Drain streams before `server.Shutdown` |

Use the [chat example](../chat) for rooms, stats and the browser UI, and
[05-usage.md](../../docs/05-usage.md) for replay, bus and presence recipes.
