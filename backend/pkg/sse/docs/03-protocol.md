# Protocol

The endpoint is a normal HTTP GET response with:

```text
Content-Type: text/event-stream; charset=utf-8
Cache-Control: no-cache, no-transform
X-Accel-Buffering: no
```

The manager flushes headers immediately and sends an initial `retry:` field.

## Event frame

```text
id: 1710000000123-4
event: message.new
data: {"id":"m-1","content":"hello"}

```

`Event.Data` is JSON. Multi-line JSON is emitted as multiple `data:` lines, as
required by SSE. `Event.Name` is optional; an empty name is the default SSE
message event. A newline in an id or event name is rejected.

`json.RawMessage` data is validated and compacted before encoding. A bare CR or
invalid JSON cannot inject a new SSE field; `EncodeComment` also normalizes CR
and CRLF. Rejected data produces an error, not a partially written event.

The server assigns an id when the caller leaves `Event.ID` empty. Broadcasts
assign one id for all recipients of that broadcast. `Client.SendEvent` assigns a
connection-local id. Clients send the last id back in `Last-Event-ID` when they
reconnect.

## Replay

Set `Config.Replay` to a `ReplayStore` that returns events after the supplied
id. The manager writes them before entering the live stream loop. The store is
application-owned; the package does not persist an event log. If replay is not
configured, `OnConnect` can inspect `client.LastEventID()` and send missed events
with the normal client API.

Replay is capped by `MaxReplayEvents` (default 100). If `Replay` is configured,
`Event.ID` must come from the application (ULID or store sequence); the manager
will not invent an ID. If the store fails, an
event cannot be encoded, an event exceeds `MaxEventBytes`, or the replay is
truncated, the client receives:

```text
event: resync
data: {"reason":"replay_truncated"}
```

`resync` means the client must fetch current state through the application's
REST history endpoint, then reconnect from a known id. It is never silently
skipped.

Replay and live delivery can overlap while the writer catches up. The client
should deduplicate by `lastEventId`; ids are not globally monotonic across pods
when an application supplies a non-monotonic value.

## Heartbeat

The writer sends `: heartbeat` comments every `HeartbeatInterval`. Comments do
not dispatch an event to the browser and keep proxies from closing an idle
stream.

## Close events

SSE has no WebSocket close frame. Before ending a stream, the manager emits:

```text
event: server_close
retry: 3000
data: {"code":1012,"reason":"service restart; reconnect with jittered backoff","retry_ms":3000}
```

| Code | Meaning |
| --- | --- |
| 1012 | Service restart; reconnect with randomized backoff. |
| 1013 | Slow client/queue backpressure; reconnect and resynchronize. |
| 4001 | Token expiry; refresh before reconnecting. |
| 4002 | Maximum connection age; reconnect with a fresh session. |

`Client.Close` uses code 1000.

For 1012/1013, the close event's `retry` is randomized between 1x and 2x
`Config.Retry` (default 3s). For 4002 it is randomized ±10%. The initial stream
`retry:` field is the base value; the close event updates the browser's next
reconnect delay. Use `Client.CloseInfo()` from `OnDisconnect` to distinguish
these reasons in metrics.

## Limits and errors

| Limit | Default | Config |
| --- | --- | --- |
| Event frame | 1 MiB | `MaxEventBytes` |
| Per-type data | optional | `EventLimits[name].MaxDataBytes` |
| Outbound queue | 256 frames | `SendQueue` |
| Slow-client escalation | 3 full queues in 30s | `SlowClientLimit`, `SlowClientWindow` |
| Replay burst | 100 events | `MaxReplayEvents` |
| Last-Event-ID input | 256 bytes | `MaxLastEventIDBytes` |
| Total/user/room connections | 10,000 / 10 / 1,000 | `MaxConnections`, `MaxConnectionsPerUser`, `MaxRoomsPerClient` |
| Auth/handshake concurrency | 10s / 128 | `AuthTimeout`, `MaxConcurrentHandshakes` |

If the identity loader returns more rooms than `MaxRoomsPerClient`, the manager
logs a warning and trims the list instead of rejecting an otherwise valid
login. Room changes from REST should use `JoinUser`/`LeaveUser`.

`EventLimits` is keyed by the SSE event name. It checks `len(Event.Data)` before
the frame is encoded and returns `ErrEventTooLarge` to the publisher. It is a
producer-side guard; SSE has no inbound event traffic to rate limit per client.

Queue overflow returns `ErrSlowClient` for that recipient. A room broadcast
continues with the other recipients and reports each missing recipient through
`OnDrop`/`Stats.DroppedEvents`; the broadcast method returns an error only when
no recipient accepted the event. This is deliberate: a partial room drop must
not stop delivery to healthy clients.

By default `QueueFullPolicy` is `QueueFullClose`: the first drop closes the
stream with 1013 before another frame is accepted, so `Last-Event-ID` cannot
advance past the missing event. `QueueFullDrop` is the legacy best-effort mode
and can create a gap; do not use it for lossless chat. See
[06-production-readiness.md](06-production-readiness.md).

429 and 503 responses include `Retry-After: 1`. Rejections are observable via
`OnReject` with reasons such as `origin`, `auth`, `token_expired`,
`handshake_limit`, `connection_limit` and `user_connection_limit`.

`OPTIONS` preflight is answered with 204 and allows `Authorization`,
`Content-Type` and `Last-Event-ID`. Prefer cookie authentication for browser
clients; tokens in query strings leak through proxy logs and history.
