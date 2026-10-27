# Runnable SSE example

A single-process demo that exercises the real `pkg/sse` API end to end:
authenticated stream setup, room and user publishing, stats, middleware and
graceful shutdown. The web page uses a real browser `EventSource`.

## Run

```bash
go run ./pkg/sse/examples/chat
```

Open <http://localhost:8090>. Set a user and a room, click **Connect**, then
publish a message. Open a second browser tab with another user in the same room
to see fanout. Set a different room to see that the event does not arrive.

Use another port with `ADDR=:8091`.

## Try it with curl

Terminal 1, keep the stream open:

```bash
curl -N "http://localhost:8090/events?user=ana&room=general"
```

Terminal 2, publish to the room:

```bash
curl -X POST "http://localhost:8090/publish/room?room=general" \
  -H "Content-Type: application/json" \
  -d "{\"text\":\"hello from curl\"}"
```

Publish to every device of one user:

```bash
curl -X POST "http://localhost:8090/publish/user?user=ana" \
  -H "Content-Type: application/json" \
  -d "{\"text\":\"private notice\"}"
```

Inspect the counters:

```bash
curl "http://localhost:8090/stats"
```

Stop the stream with `Ctrl+C`. The terminal running the server logs the
disconnect and the stream lifetime.

## Which function is called where

| Place | Call | Purpose |
| --- | --- | --- |
| `main` | `sse.New(sse.Config{...})` | Validate config and build the hub |
| `main` | `manager.Use(logStreamLifetime)` | Measure the stream lifetime |
| `main` | `manager.Start(ctx)` | Mark ready and subscribe to `Config.Bus` |
| `main` | `mux.Handle("/events", manager)` | Open the SSE stream |
| `authenticate` | `Config.Authenticate` | Return `sse.Identity` with user and rooms |
| `publishToRoom` | `manager.PublishEventToRoom` | Bus-aware room fanout |
| `publishToUser` | `manager.PublishEventToUser` | Bus-aware per-user fanout |
| `showStats` | `manager.Stats` | Connections, users, rooms, drops |
| `main` | `manager.Shutdown` | Drain streams before `server.Shutdown` |

## What is intentionally simplified

This example uses query parameters as a fake login and trusts them. A real
service must validate a session or JWT in `Authenticate` and load room
membership from the database. The example also omits `Bus`, `Presence` and
`ReplayStore`, which need infrastructure. The contracts and production
checklist are in [../../docs/05-usage.md](../../docs/05-usage.md) and
[../../INTEGRATION.md](../../INTEGRATION.md).
