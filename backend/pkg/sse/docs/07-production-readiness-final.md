# Production readiness: post-fix review

Status: 2026-10-25, after the P0.1 and P0.2 fixes. The pre-fix review is kept in
[06-production-readiness.md](06-production-readiness.md).

Method: source review, two independent re-audits, one final verifier,
`go vet ./pkg/sse/...`, `go test -race ./pkg/sse -count=1`, and statement
coverage. Coverage is 78.4%. No production code in this repository imports
`pkg/sse` yet.

## Verdict

| Profile | Score | Verdict |
| --- | --- | --- |
| Single-pod, best-effort notifications | 7.5/10 | Conditional go with strict configuration |
| Single-pod, lossless chat with replay | 6.5/10 | No-go until replay bounds and per-client ordering are fixed |
| Multi-pod | 4/10 | No-go until acknowledged, ordered, durable control delivery |
| Overall | 7/10 | Improved from 6/10 after the P0 fixes |

P0.1 and P0.2 are closed at the layers they targeted. The remaining risks are
the guarantees around ordering, replay, and acknowledged delivery, not the
basic stream path.

## P0 fixes delivered

### P0.1 Atomic slow-client gap: PASS

`sendMu` is now an exclusive `sync.Mutex`. `sendFrame` holds it across the
closing/gapped check, the pending increment, and the non-blocking enqueue
(`client.go:197`). When the queue is full and the policy is `QueueFullClose`, it
sets `gapped` and starts `gracefulClose` while still holding the lock
(`client.go:203`). No other sender can enter the queue after that point.
`Hub.deliver` marks the gap before calling `OnDrop` (`hub.go:186`, `hub.go:191`),
so a slow callback cannot reopen the window. Direct `Client.Send` and
`SendComment` use the same path and also close on a full queue. `QueueFullDrop`
is the only path that skips the gap and remains explicitly best-effort.

Regression coverage: `TestQueueFullClosesOnFirstDrop`,
`TestQueueFullGapSetBeforeOnDrop`, `TestDirectSendClosesOnFullQueue`, and
`TestSlowClientStreamClosesWith1013`. The final verifier found no deadlock or
double-close regression. `closing` uses CompareAndSwap to elect one drainer, and
`closeOnce` protects `done`.

Keep-alive coverage also includes `TestHeartbeatKeepsIdleStreamAlive` and
`TestSlowClientDropReconnectsWithReplay`. The second test drives a real queue
drop, verifies the 1013 close and the last delivered ID, then reconnects with
`Last-Event-ID` and asserts that the dropped event is replayed.

### P0.2 Cross-pod control: PASS for propagation

`JoinUser`, `LeaveUser`, `DisconnectUser` and `CloseRoom` now apply locally and
publish a control message when `Config.Bus` is set (`manager.go:472`).
`BusMessage` carries `Rooms`, `Code` and `Reason` (`bus.go:23`).
`applyBusMessage` applies every control kind without republishing, so a bus echo
cannot loop (`manager.go:598`). Applying the same control twice is safe because
the hub operations are idempotent. The two-manager tests cover remote join,
leave, disconnect and close-room (`control_test.go`).

This closes the original gap where a logout, ban, or room removal handled by one
pod never reached a socket on another pod. It does not yet guarantee delivery:
publish failures are logged and the methods return local counts only. A durable
Bus adapter or an outbox is still required for strict revocation.

## Remaining P1 before production

1. **Per-client ID ordering under concurrent publishers.** Event IDs are
   assigned before the send gate (`hub.go:166`). Two concurrent publishers can
   enqueue a higher ID and then drop a lower ID, so the client has already
   advanced past a gap. The send gate serializes enqueueing, not ID assignment.
   Assign or validate IDs under the per-client gate, or maintain a watermark and
   close on an out-of-order ID.
2. **Replay is still unbounded in time and input.** `callReplay` uses the
   manager-derived client context, not the request context (`manager.go:360`),
   and `writeReplay` truncates only after the store returns
   (`manager.go:320`). Add `ReplayTimeout`, a limit parameter or chunked API,
   apply `EventLimits` to replay, and close after `resync` or provide a recovery
   cursor.
3. **Bus acknowledgement and ordering.** `Bus.Subscribe` returns no apply error
   (`bus.go:36`), `Start` logs and continues (`manager.go:97`), and control
   publish failures are log-only (`manager.go:510`). Ack after a successful
   apply, preserve per-room order, and test a non-echoing publisher, duplicate
   echo, and a failing adapter.
4. **Queue memory is frame-count bounded.** `SendQueue` defaults to 256 frames
   at up to 1 MiB each (`config.go:16`, `client.go:93`). Add `MaxQueuedBytes`
   with atomic reserve and release on enqueue, dequeue and in-flight writes.
5. **Connection limits are per pod.** `checkCapacity` reads the local hub
   (`manager.go:691`): 10,000 and 10 per user become `10,000 x pods` and
   `10 x pods`. Document the per-pod semantics or use a shared limiter.
6. **Revocation is timer-only.** `installExpiry` installs nothing when both
   `ExpiresAt` and `MaxConnectionAge` are zero (`manager.go:793`), and
   `disconnect_user` closes every device with no session or connection ID.
   Require an expiry, add session-specific control, and add an active-stream
   logout test.
7. **Presence refresh can race removal and panic.** Every heartbeat starts a
   goroutine with a background context (`manager.go:407`); a late refresh can
   recreate a TTL entry after removal, and adapter panics are recovered
   unreliably. Use one cancelable refresh worker per client and recover around
   every adapter call.
8. **Auth timeout leaves detached goroutines.** `callAuthenticate` returns on
    timeout while the callback may still block (`manager.go:648`). Use a bounded
    worker or hold the handshake slot until the callback returns.
9. **Control delivery is best-effort.** Local counts are snapshots; async
    publish work is not tracked by shutdown; a consumer can block for
    `CloseDrainTimeout`; and an adapter that only accepts event kinds silently
    drops controls. Document the adapter contract and use a durable queue or
    outbox for strict enforcement.

Origin scheme, host, path and userinfo validation and CORS headers on
rejections were fixed after the post-fix review. See `origin.go`, `manager.go`
and `origin_test.go`.

## P2 improvements

| Area | Finding | Action |
| --- | --- | --- |
| Stats | `EventsSent` counts enqueues, not writes (`hub.go:203`) | Rename or add `FramesWritten`, queued bytes and a closed state |
| Replay | Replay bypasses `EventLimits` | Apply the same per-type guard |
| Write deadline | `SetWriteDeadline` errors are ignored (`client.go:221`) | Log or fail the stream |
| Hub leave | `leave` returns true for a non-member (`hub.go:286`) | Check membership under the lock |
| IDs | Generated IDs use clock plus local sequence | Use application IDs with Bus and Replay |
| Load test | Only the synthetic 1000-client benchmark exists | Run a real 10k-stream test through the proxy |

## What is solid

- Authentication runs before SSE headers, with explicit 401, 403, 429 and 503
  responses and `Retry-After`.
- Event encoding rejects CR/LF in `id` and `event`, compacts JSON, and splits
  multi-line data correctly.
- The per-client send gate is exclusive, the queue is bounded, and a full queue
  closes with 1013 by default before another frame can advance `Last-Event-ID`.
- Shutdown drains once, sends `server_close`, and runs presence removal and
  `OnDisconnect` off the connection wait group.
- Cross-pod join, leave, disconnect and close-room controls are implemented and
  tested with two managers.
- `go vet` passes, `go test -race` passes, and coverage is 78.4%.

## Go-live checklist

### Single-pod best-effort

- [ ] Set exact HTTPS origins and a deliberate credentials policy.
- [ ] Require `Identity.ExpiresAt` or `MaxConnectionAge`.
- [ ] Add CORS headers to rejections.
- [ ] Lower `SendQueue` and `MaxEventBytes` for the expected payloads.
- [ ] Alert on drops, 1013 closes, rejects and replay resync.

### Single-pod lossless chat

- [ ] Close the per-client ID ordering hole under concurrent publishers.
- [ ] Add `ReplayTimeout`, a bounded replay input and replay `EventLimits`.
- [ ] Add `MaxQueuedBytes` and test worst-case retention.
- [ ] Run a drop, reconnect and replay integration test end to end.

### Multi-pod

- [ ] Ack broker messages only after successful apply.
- [ ] Preserve per-room ordering and test duplicate and out-of-order delivery.
- [ ] Use a durable Bus or outbox for control messages.
- [ ] Divide connection limits by replica count or use a shared limiter.
- [ ] Add session or connection-specific revocation.

The pre-fix findings remain in [06-production-readiness.md](06-production-readiness.md)
for comparison. The operational checklist in [INTEGRATION.md](../INTEGRATION.md)
remains the deployment reference.
The current support boundary is in
[08-production-scope.md](08-production-scope.md).
