# Getting started

## Dependency

The package uses only the Go standard library. It is a normal package inside the
host module and is imported as
`github.com/AhmadKusumahDEV/go-chat/pkg/sse`.

## Mounting

```go
manager, err := sse.New(sse.Config{
	Authenticate: func(r *http.Request) (sse.Identity, error) {
		claims, err := verifyToken(r)
		if err != nil {
			return sse.Identity{}, sse.NewAuthError(http.StatusUnauthorized, "invalid token")
		}
		rooms, err := loadRooms(r.Context(), claims.UserID)
		if err != nil {
			return sse.Identity{}, err
		}
		return sse.Identity{
			UserID: claims.UserID,
			Rooms: rooms,
			Values: map[string]any{"role": claims.Role},
			ExpiresAt: claims.ExpiresAt,
		}, nil
	},
	AllowedOrigins: []string{"https://app.example.com"},
})
if err != nil {
	log.Fatal(err)
}

if err := manager.Start(ctx); err != nil {
	log.Fatal(err)
}

mux := http.NewServeMux()
mux.Handle("/events", manager)
server := &http.Server{Addr: ":8080", Handler: mux}

serverErr := make(chan error, 1)
go func() { serverErr <- server.ListenAndServe() }()

select {
case <-ctx.Done():
case err := <-serverErr:
	if !errors.Is(err, http.ErrServerClosed) {
		log.Printf("http server: %v", err)
	}
}

shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
defer cancel()

// Stop the SSE streams first. server.Shutdown waits for active connections,
// and an SSE stream is active until the manager closes it.
_ = manager.Shutdown(shutdownCtx)
_ = server.Shutdown(shutdownCtx)

/*
Alternative with the same ordering:
server.RegisterOnShutdown(func() { _ = manager.Shutdown(shutdownCtx) })
_ = server.Shutdown(shutdownCtx)
*/
```

For production, handle `ListenAndServe` errors without `log.Fatal` on the
serving path and call `manager.Shutdown` before exiting. The manager sends
`server_close` with code 1012 and a jittered-backoff hint.

## Gin and Echo

```go
router.GET("/events", gin.WrapH(manager))
e.GET("/events", echo.WrapHandler(manager))
```

## Client

```js
const source = new EventSource("/events", { withCredentials: true });
source.addEventListener("message.new", (event) => {
  const payload = JSON.parse(event.data);
  console.log(payload, event.lastEventId);
});
source.onerror = () => {
  // Automatic reconnect only covers a broken stream, not 401/403/429/503.
};
```

`EventSource` reconnects automatically only for a broken stream. According to
the EventSource specification, a non-200 response such as 401, 403, 429 or 503
is a permanent failure for that EventSource instance; it does not retry. During
a reconnect storm, a 503 from `MaxConcurrentHandshakes` would otherwise leave the
client permanently disconnected. Use an explicit reconnect loop:

```js
function connect(lastId = "") {
  let lastSeenId = lastId;
  const query = lastId ? `?last_event_id=${encodeURIComponent(lastId)}` : "";
  const source = new EventSource(`/events${query}`, { withCredentials: true });
  source.onerror = () => {
    source.close();
    // Add 1x-2x jitter and retry with capped exponential backoff.
    setTimeout(() => connect(lastId), 1000 + Math.random() * 1000);
  };
  source.addEventListener("server_close", (event) => {
    const info = JSON.parse(event.data);
    source.close();
    if (info.code === 4001) {
      refreshToken().then(() => connect(lastSeenId));
      return;
    }
    setTimeout(() => connect(lastSeenId), jitter(info.retry_ms || 3000));
  });
  source.addEventListener("message.new", (event) => {
    lastSeenId = event.lastEventId;
    // process event
  });
}
```

`ResyncEvent` (`event: resync`) means replay was incomplete. Close the stream,
fetch state from the REST history endpoint, then reconnect from the latest
known id. A 4001 token-expiry event requires `close()` and token refresh; an
automatic EventSource reconnect would just receive 401 again.
