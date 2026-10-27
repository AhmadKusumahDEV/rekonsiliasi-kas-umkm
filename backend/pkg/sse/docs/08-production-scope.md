# Production scope and support boundary

Production readiness only means something for a defined profile. This page is
the support boundary for `pkg/sse` after the P0 fixes. It prevents a deployment
from assuming lossless or multi-pod guarantees that the package does not yet
provide.

## Current decision

| Profile | Status | Allowed use |
| --- | --- | --- |
| Single pod, best-effort notifications | **Supported with conditions** | Notices, presence-style updates, non-critical fanout |
| Single pod, lossless chat with replay | **Not supported** | Wait for milestone M2 |
| Multi-pod | **Not supported** | Wait for milestone M3 |

The supported baseline is one process, authenticated streams, bounded queues,
client close on the first queue drop, and best-effort delivery. A dropped
notification may be lost, but the client is told to reconnect and a later frame
cannot silently advance `Last-Event-ID` past it.

## What bare minimum means here

A profile is at the bare minimum when all of these are true:

1. It has no open P0 finding for that profile.
2. Its failure modes are explicit: auth reject, connection limit, queue drop,
   1013 close, token expiry and shutdown.
3. Required configuration is documented and checked by the host.
4. The failure modes have tests, including the race detector.
5. The host has metrics and alerts for drops, rejects, reconnects and latency.

The single-pod best-effort profile meets items 1, 2, 4 and 5 with conditions.
Item 3 is a deployment responsibility: set strict origins, require an expiry,
and size the limits.

## Required configuration for the supported profile

| Setting | Required value | Why |
| --- | --- | --- |
| `Authenticate` | Verify signature, issuer, audience and expiry | The stream is private |
| `Identity.ExpiresAt` or `MaxConnectionAge` | At least one non-zero | An active stream is not revalidated after the handshake |
| `AllowedOrigins` | Exact HTTPS origins; no wildcard with credentials | Browser boundary |
| `QueueFullPolicy` | `QueueFullClose` (default) | Do not use `QueueFullDrop` for data that matters |
| `SendQueue` | Sized from measured memory, not the default | Default 256 frames can retain a lot of memory |
| `MaxEventBytes` / `EventLimits` | Small per event type | Protect every client queue |
| `MaxConnections` / `MaxConnectionsPerUser` | Per-pod target divided by replicas | Limits are local to one manager |
| Host `http.Server.WriteTimeout` | `0` | A fixed write timeout cuts every stream |
| Reverse proxy | No buffering, no gzip for `text/event-stream` | SSE needs prompt flushes |

## In scope

- One `*sse.Manager` per process.
- One authenticated SSE stream per connection, multiple devices per user.
- Local user, room and global fanout with bounded queues.
- Close on the first queue drop with 1013, or the explicit legacy drop mode.
- Heartbeats, graceful drain, `server_close`, token expiry and connection age.
- Best-effort `ReplayStore` support with application event IDs.
- Best-effort cross-pod control propagation through `Bus`; single-pod support
  remains the recommended deployment until M3.

## Out of scope

- Exactly-once or lossless delivery under concurrent publishers.
- Durable replay, replay timeouts, or bounded replay input.
- Acknowledged, ordered, durable multi-pod control delivery.
- Session-specific revocation; the current control closes every device.
- Global connection limits across replicas.
- A 10,000-stream capacity guarantee from the synthetic benchmark.
- Inbound client events. SSE is one-way; commands belong on REST or WebSocket.

## Milestones

### M1: single-pod best-effort (current)

Status: supported with the configuration above. Origin scheme and CORS on
rejections are fixed. Remaining host work: require an expiry, size the limits,
and configure the proxy and alerts.

### M2: single-pod lossless chat

Close these before claiming lossless delivery:

1. Per-client ID ordering under concurrent publishers.
2. `ReplayTimeout`, bounded replay input and replay `EventLimits`.
3. `MaxQueuedBytes` and a worst-case memory test.
4. Drop, reconnect and replay integration test.

### M3: multi-pod

Close these before running more than one replica:

1. Ack after successful apply and preserve per-room order.
2. Durable control delivery through an outbox or an acking adapter.
3. Shared limits or explicit per-pod math.
4. Session-specific revocation.
5. Two-instance broker test and a real proxy load test.

## Audit policy

Re-audit after a milestone, not after every small edit. A new audit before the
M2 or M3 code changes would repeat the same known findings. The pre-fix and
post-fix reports are in [06-production-readiness.md](06-production-readiness.md)
and [07-production-readiness-final.md](07-production-readiness-final.md).
