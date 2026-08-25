package signaling

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gorilla/websocket"
	"github.com/yatishydv/rtm/internal/protocol"
)

func TestSignalingServer(t *testing.T) {
	server := NewServer()
	s := httptest.NewServer(http.HandlerFunc(server.HandleWS))
	defer s.Close()

	wsURL := "ws" + strings.TrimPrefix(s.URL, "http")

	// 1. Host registers
	hostConn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("Failed to connect host: %v", err)
	}
	defer hostConn.Close()

	err = hostConn.WriteJSON(protocol.Message{Type: "register_host"})
	if err != nil {
		t.Fatalf("Failed to register host: %v", err)
	}

	var msg protocol.Message
	err = hostConn.ReadJSON(&msg)
	if err != nil {
		t.Fatalf("Failed to read host registration response: %v", err)
	}

	if msg.Type != "host_registered" || msg.Session == "" {
		t.Fatalf("Expected host_registered, got: %v", msg.Type)
	}
	sessionCode := msg.Session

	// 2. Remote connects with valid code
	remoteConn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("Failed to connect remote: %v", err)
	}
	defer remoteConn.Close()

	err = remoteConn.WriteJSON(protocol.Message{
		Type:    "connection_request",
		Session: sessionCode,
	})
	if err != nil {
		t.Fatalf("Failed to send connection request: %v", err)
	}

	// 3. Host receives connection request
	var hostMsg protocol.Message
	err = hostConn.ReadJSON(&hostMsg)
	if err != nil {
		t.Fatalf("Failed to read incoming connection on host: %v", err)
	}

	if hostMsg.Type != "incoming_connection" {
		t.Fatalf("Expected incoming_connection, got: %v", hostMsg.Type)
	}

	// 4. Host accepts connection
	err = hostConn.WriteJSON(protocol.Message{
		Type:    "connection_response",
		Session: sessionCode,
		Payload: "accepted",
	})
	if err != nil {
		t.Fatalf("Failed to accept connection: %v", err)
	}

	// 5. Remote receives acceptance
	var remoteMsg protocol.Message
	err = remoteConn.ReadJSON(&remoteMsg)
	if err != nil {
		t.Fatalf("Failed to read connection response on remote: %v", err)
	}

	if remoteMsg.Type != "connection_response" || remoteMsg.Payload != "accepted" {
		t.Fatalf("Expected accepted connection response, got: %v", remoteMsg)
	}

	// 6. Test Signal Routing
	err = hostConn.WriteJSON(protocol.Message{
		Type:    "signal",
		Session: sessionCode,
		Signal:  "test_sdp_offer",
	})
	if err != nil {
		t.Fatalf("Failed to send signal: %v", err)
	}

	var signalMsg protocol.Message
	err = remoteConn.ReadJSON(&signalMsg)
	if err != nil {
		t.Fatalf("Failed to read signal: %v", err)
	}

	if signalMsg.Type != "signal" || signalMsg.Signal != "test_sdp_offer" {
		t.Fatalf("Signal routing failed, got: %v", signalMsg)
	}

	// 7. Test Brute Force / Invalid Session
	invalidConn, _, _ := websocket.DefaultDialer.Dial(wsURL, nil)
	defer invalidConn.Close()

	for i := 0; i < 6; i++ {
		invalidConn.WriteJSON(protocol.Message{
			Type:    "connection_request",
			Session: "invalid_code",
		})
		
		var errMsg protocol.Message
		invalidConn.ReadJSON(&errMsg)
		if errMsg.Type != "error" {
			t.Fatalf("Expected error for invalid code")
		}
	}
}
