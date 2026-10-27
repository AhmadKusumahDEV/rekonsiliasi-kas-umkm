package sse

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
)

// Event is one Server-Sent Event. ID is the SSE id used for Last-Event-ID
// replay; Name is the SSE event name. Data is marshaled as JSON when it is not
// already json.RawMessage. Retry is optional and is written in milliseconds.
type Event struct {
	ID    string          `json:"id,omitempty"`
	Name  string          `json:"name,omitempty"`
	Data  json.RawMessage `json:"data,omitempty"`
	Retry time.Duration   `json:"retry,omitempty"`
}

// NewEvent marshals data into an Event.
func NewEvent(name string, data any) (Event, error) {
	payload, err := marshalData(data)
	if err != nil {
		return Event{}, err
	}
	return Event{Name: name, Data: payload}, nil
}

// EncodeEvent writes one SSE record. It rejects CR/LF in ID or Name so a
// caller cannot inject another protocol field.
func EncodeEvent(event Event) ([]byte, error) {
	if strings.ContainsAny(event.ID, "\r\n") {
		return nil, errors.New("sse: event id contains a newline")
	}
	if strings.ContainsAny(event.Name, "\r\n") {
		return nil, errors.New("sse: event name contains a newline")
	}

	var buffer bytes.Buffer
	if event.ID != "" {
		buffer.WriteString("id: ")
		buffer.WriteString(event.ID)
		buffer.WriteByte('\n')
	}
	if event.Name != "" {
		buffer.WriteString("event: ")
		buffer.WriteString(event.Name)
		buffer.WriteByte('\n')
	}
	if event.Retry > 0 {
		fmt.Fprintf(&buffer, "retry: %d\n", event.Retry.Milliseconds())
	}

	data := event.Data
	if len(data) == 0 {
		data = json.RawMessage("null")
	} else {
		var compacted bytes.Buffer
		if err := json.Compact(&compacted, data); err != nil {
			return nil, err
		}
		data = compacted.Bytes()
	}
	normalized := strings.ReplaceAll(string(data), "\r\n", "\n")
	normalized = strings.ReplaceAll(normalized, "\r", "\n")
	for _, line := range strings.Split(normalized, "\n") {
		buffer.WriteString("data: ")
		buffer.WriteString(strings.TrimSuffix(line, "\r"))
		buffer.WriteByte('\n')
	}
	buffer.WriteByte('\n')
	return buffer.Bytes(), nil
}

// EncodeComment writes an SSE comment, used for heartbeats.
func EncodeComment(text string) []byte {
	if text == "" {
		return []byte(":\n\n")
	}
	var buffer bytes.Buffer
	normalized := strings.ReplaceAll(text, "\r\n", "\n")
	normalized = strings.ReplaceAll(normalized, "\r", "\n")
	for _, line := range strings.Split(normalized, "\n") {
		buffer.WriteString(": ")
		buffer.WriteString(strings.TrimSuffix(line, "\r"))
		buffer.WriteByte('\n')
	}
	buffer.WriteByte('\n')
	return buffer.Bytes()
}

func marshalData(data any) (json.RawMessage, error) {
	if data == nil {
		return nil, nil
	}
	if raw, ok := data.(json.RawMessage); ok {
		return raw, nil
	}
	encoded, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}
	return encoded, nil
}

// CloseEvent builds the event sent before an SSE stream ends. SSE has no close
// frame, so the code and reason are carried in JSON data and the client should
// reconnect using jittered backoff for 1012/1013.
func CloseEvent(code int, reason string, retry time.Duration) Event {
	data, _ := json.Marshal(map[string]any{
		"code":     code,
		"reason":   reason,
		"retry_ms": retry.Milliseconds(),
	})
	return Event{Name: "server_close", Data: data, Retry: retry}
}

func ErrorEvent(code, message string) Event {
	data, _ := json.Marshal(map[string]string{"code": code, "message": message})
	return Event{Name: "error", Data: data}
}

// ResyncEvent tells a client that replay could not produce a complete stream
// and that it should fetch state through the application's REST history API.
func ResyncEvent(reason string) Event {
	data, _ := json.Marshal(map[string]string{"reason": reason})
	return Event{Name: "resync", Data: data}
}
