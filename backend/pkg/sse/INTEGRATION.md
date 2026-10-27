# Integration guide

This document is generic; rollout decisions belong in the host project.

## Recommended composition

| Concern | Integration | Package seam |
| --- | --- | --- |
| Auth | JWT/session/OIDC adapter | `Config.Authenticate` |
| Presence | Redis TTL set, NATS KV or database | `Config.Presence` |
| Fanout | RabbitMQ queues or Redis Streams | `Config.Bus` |
| Replay | database/event log indexed by id | `Config.Replay` |
| Metrics | `Stats`, `OnDrop`, `OnConnect`/`OnDisconnect` | application adapter |
| Reverse proxy | no response buffering, long idle timeout | deployment |
| TLS | edge termination | deployment |

## Reliability boundary

The local hub delivers to every matching live client queue. A full queue is an
explicit error and drop hook, not a silent skip. Plain Redis Pub/Sub is not a
good default fanout because it is fire-and-forget: a subscriber that misses a
message has no recovery. Use one of these patterns instead:

1. **RabbitMQ fanout/topic exchange** with one durable or exclusive per-pod
   queue. Every pod receives every event and applies it to its local hub. Ack
   only after `applyBusMessage` returns.
2. **Redis Streams consumer groups** with one group per pod, explicit
   acknowledgement and pending-entry recovery.
3. **NATS JetStream** with durable consumers.

The package `Bus` interface is broker-neutral:

```go
type Bus interface {
	Publish(ctx context.Context, message BusMessage) error
	Subscribe(ctx context.Context, handler func(BusMessage)) error
}
```

The bus must route user/room/all kinds to every subscriber that may own a
socket. The publisher's process should also receive its own message, or call the
local delivery path exactly once. Duplicate delivery is possible with durable
queues; include an application event id in `Event.Data` when the client must
deduplicate.

`Bus.Subscribe` currently returns no error from its handler. `Manager.Start`
logs an apply failure and continues, so an adapter that acknowledges on handler
return can acknowledge a message that was not delivered. Keep the message
unacknowledged until the host knows it was applied, and consume in an order
that preserves per-user and per-room delivery.

### Multi-pod control plane

When `Config.Bus` is set, `JoinUser`, `LeaveUser`, `DisconnectUser` and
`CloseRoom` apply locally and publish a control message (`join_user`,
`leave_user`, `disconnect_user`, `close_room`). Every subscriber must deliver
all kinds and apply them to its own hub. Application is idempotent, so a bus
echo may apply a control twice. A failed publish is logged and does not roll
back the local change, so a durable adapter or a host outbox is still required
for strict delivery.

## Observability and middleware

`Manager.Use` wraps the SSE HTTP handler, so a middleware can measure the
stream lifetime and add request-scoped logging/tracing. It receives
`http.Handler`; SSE has no inbound event handler. A wrapper that replaces the
`ResponseWriter` must preserve `http.Flusher`, otherwise the manager returns a
500 before committing the stream.

`OnPublish` receives the event, publish duration and error for server-side
delivery through Manager publish methods; a direct `Client.Send` bypasses it.
`OnDrop` covers per-client queue drops, and `Stats` carries the
aggregate gauges/counters. Keep these hooks fast and label metrics by event type
only if the cardinality is bounded.

## Replay and Last-Event-ID

Implement `ReplayStore` against an application event log. Return events after
`lastEventID` in ascending order. The manager writes replay events before the
live loop. Events not in the store are not magically recovered; a REST history
endpoint remains the final synchronization path. Use short event ids and avoid
placing sensitive data in ids.

`MaxReplayEvents` (default 100) bounds the replay burst. If the store fails, an
event cannot be encoded, an event exceeds `MaxEventBytes`, or the replay is
truncated, the client receives `event: resync` and must fetch state through REST
before reconnecting. This prevents a large replay from filling the live queue
and closing an otherwise healthy client.

When `Replay` is configured, `Event.ID` is mandatory for every published event.
Use a ULID or the application store's sequence; do not mix package-local client
IDs with cross-pod IDs. `Last-Event-ID` is limited by `MaxLastEventIDBytes`
before it reaches the adapter, but the adapter must validate its syntax. A
reconnecting client can receive an overlapping live event and replay event;
deduplicate by `lastEventId` on the client.

## Presence

The `Presence` interface uses connection ids so multiple devices do not
overwrite each other. `Manager.IsOnline` checks the local hub first; a remote
lookup happens only when the user is not local. Presence failures are logged and
do not reject a stream.

Implement `PresenceRefresher.Refresh` when the backend uses TTL keys. The
manager calls it on every heartbeat, which extends a live entry; after a pod
crash no refresh occurs and the TTL expires. A backend that cannot refresh TTLs
should use a connection-id set with explicit removal instead.

## Authentication timeout

`AuthTimeout` releases the handshake slot even when `Authenticate` ignores the
request context, so one stuck callback cannot hold every handshake slot. The
callback goroutine itself cannot be killed by Go; it should still honor the
context to avoid accumulating blocked goroutines and database queries.

## Reverse proxy

- Disable response buffering for `/events` (`X-Accel-Buffering: no` is set by
  the package; configure the proxy as well).
- Set read/idle timeout longer than the heartbeat interval.
- Disable gzip/compression for `text/event-stream`, or use a proxy setting that
  flushes event frames promptly.
- Keep `Cache-Control: no-cache, no-transform`.
- Check `Connection: keep-alive` behavior on HTTP/1.1; HTTP/2 does not need it.
- Test reconnect behavior through the load balancer, especially after a deploy.

## CORS, preflight and auth

The manager answers `OPTIONS` preflight with 204 and allows
`Authorization`, `Content-Type` and `Last-Event-ID`. For browser clients,
prefer an HttpOnly, SameSite cookie over a token in the query string: query
strings leak into proxy/access logs and browser history. If a token must be in a
query parameter, redact the URL in every logging layer. The TypeScript SDK's
`tokenQueryParam` is a convenience, not the preferred browser auth model.

Authentication failures are logged as Info/Debug without the token error for
4xx, and as Error with details for 5xx. Keep token parsing errors out of proxy
and application logs.

## Production readiness checklist

### Capacity

- [ ] Set total, per-user and room limits from measured capacity.
- [ ] Check file descriptors and ephemeral ports.
- [ ] Measure memory per stream: response buffers, `SendQueue`, replay/store and
      application state.
- [ ] Decide whether a write buffer pool is needed for the response layer; Go's
      HTTP server and proxy settings both affect memory.
- [ ] Load test at the expected stream count and broadcast fanout rate; record
      latency, drops, reconnects and memory.

The SSE hub stores immutable client slices per user/room, so a broadcast does
not copy the room's client list. A single RWMutex remains for index lookup;
measure lock contention before introducing sharding.

The repository's `BenchmarkHubBroadcast1000` is a synthetic fanout benchmark,
not a network load test. Use a staging host and real clients for socket count,
proxy limits, ephemeral ports and memory measurements.

SSE runs on `net/http`, so there is no gorilla read/write buffer or
`WriteBufferPool` in this package. Tune `http.Server.ReadBufferSize` and
`WriteBufferSize` in the host if needed, and account for `SendQueue` (one
pointer per buffered frame), in-flight frame bytes and the replay burst. The
package's `MaxEventBytes` and `EventLimits` cap payload size, not the host's
socket buffers.

Each active stream costs one HTTP handler goroutine, one heartbeat ticker, one
send channel of `SendQueue` slots (slice headers, not a copy of every payload),
plus the host's HTTP read/write buffers and the frame bytes currently queued or
in flight. The 10k-stream result must be measured on the target host and proxy;
the package's 1000-client benchmark measures only in-memory fanout.

### Delivery

- [ ] Every instance receives every broker event through a per-instance queue.
- [ ] Slow clients are visible through `OnDrop` and 1013 close events.
- [ ] Replay store is indexed and bounded; old events have a retention policy.
- [ ] Replay truncation/error emits `resync` and the client performs REST resync.
- [ ] Clients treat 1012/1013/4001/4002 correctly and use jittered backoff.
- [ ] A client that misses a live event can resynchronize over REST.

### Security

- [ ] `Authenticate` validates signature, issuer, audience and expiry.
- [ ] Origins are strict; credentials/CORS policy matches the client.
- [ ] Room membership is checked in REST and before joining/publishing.
- [ ] Event payloads do not leak internal error text or credentials.
- [ ] Proxy/TLS configuration does not cache or buffer private streams.

### Operations

- [ ] Metrics for connections, users, rooms, sent events, drops and rejects.
- [ ] `OnReject` reasons and 429/503 `Retry-After` behavior are tested.
- [ ] `Replay` uses stable application IDs and clients deduplicate by `lastEventId`.
- [ ] Presence implements `Refresh` when backed by TTL keys.
- [ ] Alerts for reconnects, repeated 1013, broker lag and memory growth.
- [ ] Graceful shutdown tested with active streams.
- [ ] Runbook covers broker outage, replay lag and rollback.

## Blocking close operations

`DisconnectUser` and `CloseRoom` wait for graceful drain up to
`CloseDrainTimeout`. REST logout must not hold its response for that long; use
`DisconnectUserAsync`/`CloseRoomAsync` and let the stream workers finish in the
background. `Shutdown` still waits for cleanup, but presence removal and
`OnDisconnect` run in parallel and do not extend the per-connection stream wait.

The close drain uses a signal closed when queued/in-flight frames reach zero,
not a polling ticker; the stream does not drain twice. This keeps shutdown and
room-close latency bounded without one 5ms poller per connection.

## Why SSE is simpler than WebSocket here

SSE is one-way and rides on normal HTTP GET, so there is no handshake frame
protocol, no inbound event parser and no ping/pong control frame. The transport
still needs per-client queues, heartbeats, proxy flushing and explicit close
events; those are the reliability-critical parts this package owns.

## Package layout

`pkg/sse` is a normal package inside the host Go module. It has no third-party
dependencies and does not import any other package from this repository. Tests
run from the repository root:

```bash
go test ./pkg/sse
```
