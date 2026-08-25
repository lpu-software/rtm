package protocol

type Message struct {
	Type    string `json:"type"`              // "register_host", "connection_request", "connection_response", "signal"
	Session string `json:"session,omitempty"` // The session code
	Payload string `json:"payload,omitempty"` // SDP or ICE or other payload
	Signal  string `json:"signal,omitempty"`  // WebRTC Signal data
}
