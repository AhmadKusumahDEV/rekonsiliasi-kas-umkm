# Roadmap and backlog

This file is generic; host-specific rollout belongs outside the package.

## Delivered

- [x] Authenticated SSE GET stream with origin checks.
- [x] Per-user and per-room local hub delivery; multiple devices per user.
- [x] One bounded outbound queue per stream and explicit slow-client handling.
- [x] Heartbeat comments, immediate header flush and write deadlines.
- [x] Graceful close drain and `server_close` events.
- [x] `JoinUser`, `LeaveUser`, `CloseRoom`, `DisconnectUser`.
- [x] `Manager.Use` HTTP middleware and `OnPublish` duration/error hook.
- [x] Per-type `MaxDataBytes` through `EventLimits`.
- [x] `Last-Event-ID`, optional `ReplayStore`, `Client.Context`.
- [x] Stable application IDs required when replay is configured; Last-Event-ID length limit and client dedupe guidance.
- [x] Presence TTL refresh through `PresenceRefresher`, CORS preflight, async disconnect/close-room variants.
- [x] Optional `Bus` and `Presence`; `LocalBus` for local/tests.
- [x] Connection, room, auth, event-size and age limits.
- [x] Shutdown code 1012 and token/age codes 4001/4002.
- [x] Unit and loopback integration tests.
- [x] Copy-on-write user/room/global client slices; broadcasts do not copy a room's client list.
- [x] Signal-based close drain (no 5ms poller) and single drain path; presence/OnDisconnect run in parallel off connWG.
- [x] `goleak` tests, fuzz coverage for event encoding, and synthetic 1000-client fanout benchmark.
- [x] Token/age, jitter, CR injection, replay failure/oversize/truncation, slow-stream 1013, OnConnect rejection, auth-ignore-context, http.Server shutdown and no-flush tests.
- [x] `QueueFullClose` closes on the first dropped frame so `Last-Event-ID` cannot skip it; `QueueFullDrop` keeps the legacy best-effort mode.
- [x] Cross-pod control messages for join, leave, disconnect and close-room when `Bus` is configured, with two-manager tests.

Inbound per-client/per-type rate limiting is not applicable to SSE because the
client does not send events over the stream. Outbound per-client backpressure,
slow-client escalation and per-type size limits are implemented instead.

## Open work

| Priority | Item | Reason |
| --- | --- | --- |
| P1 | Per-client ordered event IDs | Concurrent publishers can enqueue a higher ID and then drop a lower ID; assign or validate IDs under the send gate. |
| P1 | Durable control delivery | Control publish failures are logged and local counts are snapshots; use an outbox or an acking adapter for strict revocation. |
| P1 | Bus ack and ordering contract | The subscribe handler cannot return an error, and per-room ordering is not guaranteed. |
| P1 | Replay timeout and byte-based queue cap | A hung replay store and frame-count memory limits are not bounded by time or bytes. |
| P1 | Broker adapter in the host project | The package exposes `Bus`; queue naming, ack, retry and dead-letter policy are deployment-specific. |
| P1 | Replay store in the host project | Retention, indexing and access control need the application's storage. |
| P2 | Sharded hub | Copy-on-write membership already removes per-broadcast snapshot copies. A single RWMutex remains; measure lock contention under real fanout before changing room/user semantics. |
| P2 | 10k+ stream load test | Requires a staging host, proxy tuning and a load generator; unit tests cannot provide a capacity guarantee. |
| P2 | Compression tuning | SSE is usually small; proxy/compression buffering can hurt latency. Benchmark before enabling. |
| P3 | Built-in metrics adapter | Keep Prometheus/OpenTelemetry out of the reusable package; hooks and `Stats` are the seam. |

## Non-goals

- Bidirectional messaging; use WebSocket for that.
- Persisting events or guaranteeing replay without `ReplayStore`.

## Synthetic benchmark

`go test ./pkg/sse -run '^$' -bench 'BenchmarkHubBroadcast1000' -benchmem`
measures a 1000-client in-memory fanout with a drain goroutine per client. On
the development machine it is roughly 254 µs/op with about 505 B and 8
allocations per broadcast. This validates the copy-on-write broadcast path; it
is not a network or 10k-connection capacity result.
- Because a broker event was published, assuming a browser that was offline
  received it; durable storage and REST resync are separate concerns.
