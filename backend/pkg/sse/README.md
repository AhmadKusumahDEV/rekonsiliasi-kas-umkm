# pkg/sse

A reusable Server-Sent Events transport for Go services. It provides
authenticated stream setup, a local user/room hub, one bounded queue per client,
graceful drain, slow-client detection and explicit reconnect events.

This is a normal package inside the host Go module. It imports only the standard
library and does not depend on any other package in this repository. Import it with
`github.com/AhmadKusumahDEV/rekonsiliasi-kas-umkm/backend/pkg/sse`, or copy the folder into the
destination module.

For the multi-process delivery model and the reason plain Redis Pub/Sub is not
the default, see [INTEGRATION.md](INTEGRATION.md).

## Start here

- Run `go run ./pkg/sse/examples/minimal` for the smallest working server.
- Read [05-usage.md](docs/05-usage.md) for the call order, feature recipes and
  best practices.
- Use [examples/chat](examples/chat) when you need rooms, stats and a browser UI.
- Check [07-production-readiness-final.md](docs/07-production-readiness-final.md)
  before choosing a production profile.

## Quick start

```go
manager, err := sse.New(sse.Config{
	Authenticate: func(r *http.Request) (sse.Identity, error) {
		claims, err := verifyToken(r)
		if err != nil {
			return sse.Identity{}, sse.NewAuthError(http.StatusUnauthorized, "invalid token")
		}
		rooms, err := loadRooms(r.Context(), claims.UserID)
		if err != nil {
			return sse.Identity{}, err // 500, not a token failure
		}
		return sse.Identity{UserID: claims.UserID, Rooms: rooms, ExpiresAt: claims.ExpiresAt}, nil
	},
	AllowedOrigins: []string{"https://app.example.com"},
})
if err != nil {
	log.Fatal(err)
}

if err := manager.Start(ctx); err != nil {
	log.Fatal(err)
}
defer manager.Shutdown(context.Background())

mux := http.NewServeMux()
mux.Handle("/events", manager)
```

The browser uses:

```js
const source = new EventSource("/events", { withCredentials: true });
source.addEventListener("message.new", (event) => {
  console.log(JSON.parse(event.data), event.lastEventId);
});
source.addEventListener("server_close", (event) => {
  const info = JSON.parse(event.data);
  source.close(); // control reconnect yourself; see docs/02
});
```

Server-side publish:

```go
_, err := manager.SendEventToUser(userID, sse.Event{
	Name: "message.new",
	Data: messageJSON,
})
_, err = manager.BroadcastToRoom(roomID, sse.Event{
	Name: "message.new",
	Data: messageJSON,
})
```

## Documented behaviour

- Every stream has `Content-Type: text/event-stream`, no-cache headers and an
  immediate flush.
- A heartbeat comment is sent every `HeartbeatInterval`.
- Each stream gets its own outbound queue. A full queue is reported through
  `ErrSlowClient`/`OnDrop`; `SendEventToUser` returns an error only when no
  device accepted the event. Partial drops are visible through `OnDrop` and
  `Stats.DroppedEvents`. The default `QueueFullPolicy` is `QueueFullClose`: the
  stream closes on the first drop, so a later frame cannot advance
  `Last-Event-ID` past the missing
  one. `QueueFullPolicy: QueueFullDrop` keeps the legacy best-effort behavior
  and is not lossless.
- Per-user delivery reaches every device. Room delivery reaches every local
  socket in the room; `exceptUserID` excludes all devices of that user.
- `JoinUser`, `LeaveUser`, `DisconnectUser` and `CloseRoom` update local
  sockets and publish a control message when `Config.Bus` is set. A Bus
  adapter must deliver every `BusMessage` kind, not only events. Applying a
  control twice is safe because the operations are idempotent.
- `Client.Close` drains queued events up to `CloseDrainTimeout` before the final
  `server_close` event. `Shutdown` uses code 1012; the event's `retry` is
  randomized between 1x and 2x `Retry`, and connection-age retry gets ±10%.
- `Event.ID` is the SSE `id`; `Last-Event-ID` is exposed on the client and can be
  passed to `ReplayStore` to recover missed events after reconnect. Replay is
  capped by `MaxReplayEvents`; an incomplete replay sends a `resync` event.
  When `Replay` is configured, every published `Event` must have an application
  ID (ULID or store sequence); package-generated IDs are refused because they
  are not stable across processes/reconnects. Deduplicate replay/live overlap
  in the client by `lastEventId`.
- `Last-Event-ID` is limited by `MaxLastEventIDBytes` before reaching the replay
  adapter. The adapter must still validate the ID syntax/content.
- `OnReject` reports `not_ready`, `origin`, `handshake_limit`, `auth`,
  `token_expired`, `connection_limit`, `user_connection_limit` and related
  reasons. 429/503 responses include `Retry-After: 1`.
- `Client.CloseInfo()` exposes the close code/reason to `OnDisconnect`.
- `PresenceRefresher.Refresh` is called on heartbeats so a TTL-backed presence
  entry stays alive; after a pod crash it expires normally.
- `DisconnectUserAsync` and `CloseRoomAsync` start closing without waiting for
  `CloseDrainTimeout`. The blocking variants remain available.
- `Bus` and `Presence` are optional. Nil keeps the hub local.
- `Manager.Use` wraps the long-lived SSE HTTP handler for logging, metrics and
  tracing with `func(Handler) Handler`; `Handler` is an alias for
  `http.Handler`. SSE has no inbound event handler. `OnPublish` observes
  server-side publish duration/error. It covers Manager publish methods; a
  direct `Client.Send` bypasses it.
- `Config.EventLimits` adds a per-event-type `MaxDataBytes` guard. There is no
  inbound per-client rate limit because SSE is one-way; outbound backpressure
  is handled by each client's queue, `OnDrop` and 1013 close events.
- SSE uses the standard `id` field and `Last-Event-ID` for ordering/gap
  detection. Websocket-style `ts`/`seq` fields are not part of the SSE event
  grammar; put application sequence data in `data` when needed.

## Documentation

- [Overview](docs/01-overview.md)
- [Getting started](docs/02-getting-started.md)
- [Protocol](docs/03-protocol.md)
- [Usage recipes and call order](docs/05-usage.md)
- [Canonical protocol](PROTOCOL.md)
- [Integration and production checklist](INTEGRATION.md)
- [Roadmap](ROADMAP.md)
- [Testing](docs/04-testing.md)
- [Pre-fix production review](docs/06-production-readiness.md)
- [Post-fix production readiness](docs/07-production-readiness-final.md)
- [Production scope and support boundary](docs/08-production-scope.md)
- [Runnable chat example](examples/chat)

## Verification

```bash
go test ./pkg/sse -count=1
go test -race ./pkg/sse -count=1
golangci-lint run ./pkg/sse/...
```
