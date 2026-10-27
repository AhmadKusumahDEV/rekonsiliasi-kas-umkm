# Overview

`pkg/sse` is a one-way, long-lived HTTP stream transport. The application owns
authentication, authorization, persistence, replay storage and cross-process
fanout. The package owns:

- authentication and origin checks before headers are committed,
- the local user/room hub,
- one bounded outbound queue per stream,
- heartbeat comments, write deadlines and flushing,
- slow-client detection and explicit close events,
- `Last-Event-ID` capture and optional replay,
- connection limits, token/age expiry, statistics and shutdown.

## Stream flow

```text
GET /events
  -> method, origin, handshake slot
  -> Authenticate with AuthTimeout
  -> connection/room limits
  -> register client in Hub
  -> join Identity.Rooms, copy Values
  -> OnConnect
  -> write SSE headers + retry, flush
  -> optional Replay(lastEventID)
  -> writer loop: queue frames, heartbeat, request cancellation
  -> on close: drain, write server_close, unregister, OnDisconnect
```

There is no inbound event processor. A client that needs to send commands uses
REST or another channel. This keeps SSE deterministic and avoids a second
protocol on the stream.

## Delivery model

`Hub.SendEventToUser`, `BroadcastToRoom` and `BroadcastToRoomExcept` snapshot
matching local clients, encode one SSE frame, and enqueue it into every matching
client queue. A slow client does not block the loop; it is counted, reported
through `OnDrop`, and eventually closed. This is intentionally different from a
fire-and-forget broker subscription where a listener that missed an event has no
way to know.

The hub is local to one process. `Bus` is the optional cross-process seam; see
[../INTEGRATION.md](../INTEGRATION.md) for the fanout and replay model.

## Feature mapping for a one-way transport

| Websocket-style feature | SSE equivalent |
| --- | --- |
| Event handler middleware | `Manager.Use(func(Handler) Handler)` around the SSE HTTP handler; `Handler` aliases `http.Handler` |
| `JoinUser`/`LeaveUser`/`CloseRoom` | Implemented on the SSE hub |
| `OnEvent(type, duration, error)` | `OnPublish(event, duration, error)`; SSE has no inbound event |
| Per-client/per-type inbound rate limit | Not applicable to a one-way stream; outbound queues/backpressure and `EventLimits.MaxDataBytes` are used |
| `Client.Context()` | Implemented |
| `ts`/`seq` envelope fields | SSE `id` and `Last-Event-ID`; application sequence can live in `data` |
| Restart code 1012 + jitter | Implemented in `server_close` and retry jitter |
| `Bus`/`Presence` | Implemented as optional interfaces |
