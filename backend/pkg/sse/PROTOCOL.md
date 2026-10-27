# SSE protocol

The canonical SSE wire contract is maintained in
[docs/03-protocol.md](docs/03-protocol.md). It covers the event stream,
`Last-Event-ID`, `resync`, close events, replay limits and the one-way delivery
model.

The server endpoint is a long-lived `GET` response with:

```text
Content-Type: text/event-stream; charset=utf-8
Cache-Control: no-cache, no-transform
X-Accel-Buffering: no
```

Events use the standard `id`, `event`, `data` and `retry` fields. Client
commands use REST or another channel; they are not sent over the SSE stream.
