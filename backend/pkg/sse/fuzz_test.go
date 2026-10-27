package sse

import (
	"encoding/json"
	"testing"
)

func FuzzEncodeEvent(f *testing.F) {
	f.Add([]byte(`{"id":"message-1","content":"hello"}`))
	f.Add([]byte(`{"text":"line\nline","n":1}`))
	f.Add([]byte(`null`))
	f.Add([]byte(`{`))
	f.Add([]byte("{\r\n\"x\":1}"))

	f.Fuzz(func(t *testing.T, data []byte) {
		_, _ = EncodeEvent(Event{
			ID:   "1",
			Name: "message.new",
			Data: json.RawMessage(data),
		})
	})
}
