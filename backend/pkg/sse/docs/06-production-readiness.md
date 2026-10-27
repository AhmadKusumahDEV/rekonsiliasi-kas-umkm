# Production readiness review

> Post-fix status: both P0 items were implemented and passed a final verifier.
> See [07-production-readiness-final.md](07-production-readiness-final.md) for
> the current verdict. The P0 sections below remain as the pre-fix record.

Scope: `pkg/sse` as a reusable transport, reviewed against single-pod and
multi-pod deployments. Method: source review of `client.go`, `hub.go`,
`manager.go`, `bus.go`, `event.go`, `config.go` and `origin.go`; read-only
checks with `go vet ./pkg/sse` and `go test -race ./pkg/sse -count=1`;
statement coverage was 73.2%. The checks passed. Findings marked as
hypotheses depend on the host adapter and must be verified in staging.

## Verdict

| Deployment profile | Verdict | Main reason |
| --- | --- | --- |
| Single pod, best-effort notifications | Conditionally ready | The stream path is sound; finish the P1 hardening below |
| Single pod, lossless chat with `Replay` | Not ready | A dropped frame can advance `Last-Event-ID` past the missing event |
| Multi-pod | Not ready | Membership, disconnect and close-room controls are pod-local |

The package is a strong transport foundation: authentication runs before SSE
headers, event encoding rejects newline injection, each stream has a bounded
queue, and shutdown has an explicit drain path. It is not yet a complete
production control plane for chat or for a multi-replica deployment.

## P0 blockers

### P0.1 A slow-client drop is invisible to replay

`Client.sendFrame` returns `ErrSlowClient` when the queue is full
(`client.go:196`). `Hub.deliver` reports the drop and continues sending later
events (`hub.go:157`). The stream is closed only after `SlowClientLimit` drops
inside the window.

Failure: a client receives `id=10`, event `11` is dropped, event `12` is
accepted. The browser now reports `Last-Event-ID: 12`. On reconnect,
`ReplayStore.Replay(..., "12")` returns events after 12, so event 11 is gone
and the client was never told to resync. With no `Replay`, the client still
cannot detect the gap from the stream.

This breaks the lossless chat use case. The existing `TestSlowClientDropHook`
asserts that the stream stays open before the threshold, so the current
behavior is intentional but unsafe for replay.

Recommended fix: on the first `ErrSlowClient`, stop accepting new frames for
that client under `sendMu`, set a gap flag, and close with 1013 after draining;
or enqueue a `resync` event once there is space and close. `Last-Event-ID` must
never advance past a frame the client did not receive. Keep the three-strike
best-effort policy only as an explicit option. Add a test that drops one event,
delivers a later one, reconnects with replay, and verifies recovery.

### P0.2 Multi-pod control operations are pod-local

`JoinUser`, `LeaveUser`, `DisconnectUser` and `CloseRoom` only touch the local
hub (`manager.go:471-486`, `hub.go:296`). `BusMessage` carries event delivery only
(`bus.go:6`). If pod A handles a logout, ban or membership removal while the
user is connected to pod B, pod B keeps the stream and the old rooms until the
token expires or the client reconnects. Room fanout can continue to reach a
removed member.

Recommended fix: either deploy exactly one pod, or add control messages for
join, leave, disconnect-user, disconnect-session and close-room, and apply them
on every pod with acknowledgement. Add a two-manager test. See the note in
[INTEGRATION.md](../INTEGRATION.md).

## P1 before scale

1. **Bus acknowledgement cannot observe apply failures.** `Bus.Subscribe` takes
   `func(BusMessage)` with no error (`bus.go:29`). `Start` logs an apply error
   and returns (`manager.go:96`), so an adapter that acknowledges after the
   handler returns can ack an oversized, malformed or missing-ID message that
   was never delivered. Return an error from the handler, or add an `OnBusError`
   contract with nack and dead-letter behavior.
2. **No ordering guarantee across bus consumers.** `applyBusMessage` delivers
   each arrival immediately (`manager.go:528`). Concurrent consumers or a retry
   after later messages can reorder a room. Partition by user or room with one
   in-flight message, or carry a sequence and reorder before delivery.
3. **Connection limits are per pod.** `checkCapacity` reads the local hub
   (`manager.go:641`). A limit of 10,000 and 10 per user becomes
   `10,000 x replicas` and `10 x replicas`. Divide the global target by the
   replica count or use a shared limiter, and document the semantics.
4. **Revocation depends on absolute expiry.** `installExpiry` installs nothing
   when `Identity.ExpiresAt` and `MaxConnectionAge` are zero (`manager.go:743`).
   Streams are not revalidated after the handshake. Require one of the two in
   production, and wire logout or ban to `DisconnectUser` on every pod.
5. **Queue memory is bounded by frame count, not bytes.** `SendQueue` defaults
   to 256 frames (`config.go:16`) and `MaxEventBytes` to 1 MiB. A slow client
   can retain up to roughly 256 MiB of distinct frames. Add `MaxQueuedBytes`
   with atomic accounting, count replay and in-flight bytes, and use a smaller
   default.
6. **Replay has no timeout and truncates after the store returns.**
   `ReplayStore.Replay` returns an unbounded slice (`config.go:122`);
   `writeReplay` slices it afterward (`manager.go:319`). A hung store blocks
   setup after headers and a huge result allocates before truncation. Add
   `ReplayTimeout`, a limit parameter or a chunked API, and emit `resync` on
   timeout.
7. **Presence refresh can race removal and panic.** Every heartbeat starts a
   new goroutine with `context.Background()` (`manager.go:406`,
   `manager.go:806`). A late refresh can recreate a TTL entry after
   `removePresence` (`manager.go:784`). A panic in the adapter is not recovered
   on that goroutine. Use one in-flight refresh per client tied to `client.ctx`,
   cancel and wait before removal, and recover around adapter calls.
8. **Auth timeout leaves the callback goroutine running.** `callAuthenticate`
   returns on `AuthTimeout` while the callback may still block
   (`manager.go:592`). A stuck auth backend accumulates one goroutine and one
   query per timed-out handshake. Use a bounded worker pool that is not released
   until the callback returns, or track concurrent detached work.
9. **A failed bus subscription can strand clients.** `Start` sets `started`
   before `Bus.Subscribe` (`manager.go:92`). If subscribe fails, it cancels and
   sets `closing`, but does not close clients. Later `Shutdown` only waits
   (`manager.go:421`). Close the current clients through the normal drain path
   before returning the subscribe error.
10. **Default origin validation ignores the scheme.** `checkOrigin` compares
    only `parsed.Host` with `r.Host` (`origin.go:24`). An HTTP origin can match
    an HTTPS host. Compare normalized scheme, host and port, and test the
    HTTP-to-HTTPS and port variants.
11. **Rejections omit CORS headers.** `reject` writes the status without
    `Access-Control-Allow-Origin` (`manager.go:722`), so a browser sees a CORS
    failure instead of the 401, 403, 429 or 503. Add the same CORS headers used
    by the stream and preflight path.
12. **Replay bypasses per-type limits.** `writeReplay` checks the encoded frame
    size but not `EventLimits` (`manager.go:319`). Apply the same per-type data
    check to replay.
13. **Write deadline errors are ignored.** `writeFrame` discards the result of
    `SetWriteDeadline` (`client.go:220`). A middleware that preserves `Flusher`
    but not deadline support can let a write hang. Log or fail the stream, and
    document the `Unwrap` requirement.
14. **Resync leaves the stream live without a cursor.** `writeReplay` sends
    `resync` and continues the live loop (`manager.go:329`). A client that is
    still far behind can repeat the loop. Close after `resync`, or include a
    head cursor and require the client to reconnect from the REST cursor.
15. **Generated IDs are not cross-pod unique.** `Hub.nextEventID` uses
    `UnixNano` plus a process-local sequence (`hub.go:205`). Use application IDs
    when `Bus` or `Replay` is configured, and document the local-only methods.

## P2 improvements

| Area | Finding | Action |
| --- | --- | --- |
| Publish with `Bus` | The publisher does not apply locally; delivery depends on broker echo (`manager.go:528`) | Document the echo requirement and test it with a two-instance fake |
| Stats | `EventsSent` counts successful enqueues, not writes (`hub.go:201`) | Rename or add a `FramesWritten` counter |
| Counters | No bus, replay, presence or handshake saturation counters | Add counters and alerts |
| Cache headers | `no-cache, no-transform` lacks `private` and `Vary` (`manager.go:298`) | Add `private`, `no-store` where appropriate, and `Vary: Authorization, Cookie` |
| Hub leave | `leave` returns true for a non-member (`hub.go:279`) | Check membership under the lock |
| Dead errors | `ErrRoomLimit` and `RejectRoomLimit` are unused | Use them or remove them |
| Example | The demo uses query identities and no bus or replay | Keep it labeled as demo code, and add a production wiring example later |

## What is already solid

- Authentication runs before SSE headers, with explicit 401, 403, 429 and 503
  responses and `Retry-After`.
- Event encoding rejects CR/LF in `id` and `event`, compacts JSON and splits
  multi-line data correctly (`event.go:33`).
- Each stream has its own bounded queue; one slow client cannot consume another
  client queue.
- Shutdown drains once, sends `server_close` with a jittered retry hint, and
  runs presence removal and `OnDisconnect` in parallel.
- Replay truncation emits `resync` instead of silently leaving a gap; the
  `Last-Event-ID` length is bounded.
- No data race or lock-order inversion was found in the review; the race suite
  passes.

## Go-live checklist

- [ ] Choose the deployment profile. Start with one pod unless P0.2 is fixed.
- [ ] Fix P0.1 and add the drop, reconnect and replay regression test.
- [ ] Require `Identity.ExpiresAt` or `MaxConnectionAge`; wire logout and ban
      to every pod.
- [ ] Use strict scheme-aware origins and add CORS headers to rejections.
- [ ] Set byte-based queue limits and small per-type event limits.
- [ ] Add `ReplayTimeout` and a bounded, shared replay store.
- [ ] Make bus consumption ordered per user or room, ack after successful
      apply, and test two instances.
- [ ] Make presence refresh single-flight, context-bound and panic-safe.
- [ ] Load test at the target stream count and fanout rate through the real
      proxy.
- [ ] Alert on drops, rejects, 1013 closes, replay resync and broker lag.

The deployment checklist in [INTEGRATION.md](../INTEGRATION.md) remains the
operational reference. This review records the gaps that must be closed before
the package is used for chat delivery at scale.
