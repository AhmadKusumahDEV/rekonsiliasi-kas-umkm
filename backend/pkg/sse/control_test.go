package sse

import (
	"net/url"
	"testing"
	"time"
)

func busTestConfig(bus Bus) Config {
	cfg := testConfig()
	cfg.Bus = bus
	return cfg
}

func TestBusControlJoinsAndLeavesRemoteClients(t *testing.T) {
	bus := NewLocalBus()
	m1 := startTestManager(t, busTestConfig(bus))
	m2 := startTestManager(t, busTestConfig(bus))
	server := serveTestManager(t, m2)

	resp, reader := openStream(t, server, url.Values{"user": {"remote"}, "room": {"base"}}, nil)
	defer resp.Body.Close()
	waitUntil(t, time.Second, func() bool { return m2.hub.RoomClientCount("base") == 1 })

	if joined := m1.JoinUser("remote", "extra"); joined != 0 {
		t.Fatalf("m1 JoinUser = %d, want 0 local sockets", joined)
	}
	waitUntil(t, time.Second, func() bool { return m2.hub.RoomClientCount("extra") == 1 })

	if sent, err := m2.BroadcastToRoom("extra", Event{Name: "remote.join"}); err != nil || sent != 1 {
		t.Fatalf("broadcast after remote join = %d, %v", sent, err)
	}
	if event := readSSEEvent(t, reader); event["event"] != "remote.join" {
		t.Fatalf("join event = %+v", event)
	}

	if removed := m1.LeaveUser("remote", "extra"); removed != 0 {
		t.Fatalf("m1 LeaveUser = %d, want 0 local sockets", removed)
	}
	waitUntil(t, time.Second, func() bool { return m2.hub.RoomClientCount("extra") == 0 })
	if sent, err := m2.BroadcastToRoom("extra", Event{Name: "after.leave"}); err != nil || sent != 0 {
		t.Fatalf("broadcast after remote leave = %d, %v", sent, err)
	}
}

func TestBusControlDisconnectsRemoteUser(t *testing.T) {
	bus := NewLocalBus()
	m1 := startTestManager(t, busTestConfig(bus))
	m2 := startTestManager(t, busTestConfig(bus))
	server := serveTestManager(t, m2)

	resp, reader := openStream(t, server, url.Values{"user": {"remote"}}, nil)
	defer resp.Body.Close()
	waitUntil(t, time.Second, func() bool { return m2.Online("remote") })

	if count := m1.DisconnectUserAsync("remote", CloseTokenExpired, "revoked"); count != 0 {
		t.Fatalf("m1 DisconnectUserAsync = %d, want 0 local sockets", count)
	}
	event := readSSEEvent(t, reader)
	assertCloseCode(t, event, CloseTokenExpired)
	waitUntil(t, time.Second, func() bool { return !m2.Online("remote") })
}

func TestBusControlClosesRemoteRoom(t *testing.T) {
	bus := NewLocalBus()
	m1 := startTestManager(t, busTestConfig(bus))
	m2 := startTestManager(t, busTestConfig(bus))
	server := serveTestManager(t, m2)

	resp, reader := openStream(t, server, url.Values{"user": {"remote"}, "room": {"closing"}}, nil)
	defer resp.Body.Close()
	waitUntil(t, time.Second, func() bool { return m2.hub.RoomClientCount("closing") == 1 })

	if count := m1.CloseRoomAsync("closing", 4000, "room closed"); count != 0 {
		t.Fatalf("m1 CloseRoomAsync = %d, want 0 local sockets", count)
	}
	event := readSSEEvent(t, reader)
	assertCloseCode(t, event, 4000)
}
