package sse

import "net/http"

// Handler is the long-lived SSE HTTP handler seam. It is an alias for
// http.Handler; SSE has no inbound event handler.
type Handler = http.Handler

// Middleware wraps the long-lived SSE HTTP handler. The first middleware
// registered is the outermost wrapper. A middleware that wraps the
// ResponseWriter must preserve http.Flusher, otherwise streaming is rejected.
type Middleware func(Handler) Handler
