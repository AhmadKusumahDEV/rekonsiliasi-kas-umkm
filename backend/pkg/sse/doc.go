// Package sse provides a reusable Server-Sent Events transport for Go services.
//
// Each stream is authenticated before headers are written, has its own bounded
// outbound queue, and is indexed in an in-memory hub by user and room. A
// broadcast enqueues the event for every matching local stream, so one client
// cannot consume another client's queue. Slow streams are reported through
// OnDrop and eventually closed with an explicit server_close event.
//
// Wiring order:
//
//	manager, err := sse.New(sse.Config{
//		Authenticate: func(r *http.Request) (sse.Identity, error) {
//			return sse.Identity{UserID: userID, Rooms: roomIDs}, nil
//		},
//	})
//	if err != nil {
//		log.Fatal(err)
//	}
//	if err := manager.Start(ctx); err != nil {
//		log.Fatal(err)
//	}
//	defer manager.Shutdown(context.Background())
//	http.Handle("/events", manager)
//
// Publish an event:
//
//	manager.SendEventToUser(userID, sse.Event{Name: "message.new", Data: payload})
//	manager.BroadcastToRoom(roomID, sse.Event{Name: "typing", Data: payload})
//
// SSE is one-way. The package does not parse inbound events; REST or another
// channel handles client commands. It supports Last-Event-ID and an optional
// ReplayStore for reconnects. For several processes, implement Bus with a
// broker; see INTEGRATION.md for the fanout and replay model.
package sse
