package signaling

import (
	"log"
	"net/http"
	"sync"
	"crypto/rand"
	"encoding/hex"

	"time"

	"github.com/gorilla/websocket"
	"github.com/yatishydv/rtm/internal/protocol"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

type Session struct {
	HostConn    *websocket.Conn
	RemoteConn  *websocket.Conn
	CreatedAt   time.Time
	FailedTries int
}

type Server struct {
	mu       sync.Mutex
	sessions map[string]*Session
}

func NewServer() *Server {
	return &Server{
		sessions: make(map[string]*Session),
	}
}

func (s *Server) HandleWS(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("Upgrade error:", err)
		return
	}
	defer conn.Close()

	var sessionCode string

	for {
		var msg protocol.Message
		if err := conn.ReadJSON(&msg); err != nil {
			log.Println("Read error:", err)
			break
		}

		switch msg.Type {
		case "register_host":
			b := make([]byte, 4)
			rand.Read(b)
			sessionCode = hex.EncodeToString(b)
			
			s.mu.Lock()
			s.sessions[sessionCode] = &Session{
				HostConn:  conn,
				CreatedAt: time.Now(),
			}
			s.mu.Unlock()

			conn.WriteJSON(protocol.Message{
				Type:    "host_registered",
				Session: sessionCode,
			})
			log.Printf("Host registered with code: %s", sessionCode)

		case "connection_request":
			reqSession := msg.Session
			s.mu.Lock()
			session, ok := s.sessions[reqSession]
			
			if ok {
				if time.Since(session.CreatedAt) > 2*time.Hour {
					ok = false
					delete(s.sessions, reqSession)
				} else if session.FailedTries > 5 {
					ok = false
				} else {
					session.RemoteConn = conn
				}
			}
			s.mu.Unlock()

			if ok {
				sessionCode = reqSession // Bind this connection to the session
				session.HostConn.WriteJSON(protocol.Message{
					Type: "incoming_connection",
					Session: reqSession,
				})
			} else {
				if session != nil {
					s.mu.Lock()
					session.FailedTries++
					s.mu.Unlock()
				}
				conn.WriteJSON(protocol.Message{
					Type: "error",
					Payload: "Invalid or expired session code",
				})
			}
		
		case "connection_response":
			s.mu.Lock()
			session, ok := s.sessions[msg.Session]
			s.mu.Unlock()

			if ok && session.RemoteConn != nil {
				session.RemoteConn.WriteJSON(msg)
			}
			
		case "signal":
			s.mu.Lock()
			session, ok := s.sessions[msg.Session]
			var target *websocket.Conn
			if ok {
				if session.HostConn == conn {
					target = session.RemoteConn
				} else {
					target = session.HostConn
				}
			}
			s.mu.Unlock()

			if target != nil {
				target.WriteJSON(msg)
			}
		}
	}

	// Cleanup
	s.mu.Lock()
	if sessionCode != "" {
		delete(s.sessions, sessionCode)
	}
	s.mu.Unlock()
}
