package webrtc

import (
	"encoding/json"
	"log"

	"github.com/pion/webrtc/v4"
)

type Peer struct {
	Connection    *webrtc.PeerConnection
	DataChannel   *webrtc.DataChannel
	OnSignal      func(signal string)
	OnMessage     func(msg string)
	OnBinary      func(data []byte)
	OnDataChannel func(d *webrtc.DataChannel)
}

func NewPeer(isHost bool, onSignal func(string), onMessage func(string), onBinary func([]byte)) (*Peer, error) {
	config := webrtc.Configuration{
		ICEServers: []webrtc.ICEServer{
			{URLs: []string{"stun:stun.l.google.com:19302"}},
		},
	}

	pc, err := webrtc.NewPeerConnection(config)
	if err != nil {
		return nil, err
	}

	p := &Peer{
		Connection: pc,
		OnSignal:   onSignal,
		OnMessage:  onMessage,
		OnBinary:   onBinary,
	}

	pc.OnICECandidate(func(c *webrtc.ICECandidate) {
		if c == nil {
			return
		}
		b, err := json.Marshal(c.ToJSON())
		if err == nil {
			p.OnSignal(string(b))
		}
	})

	setupDataChannel := func(d *webrtc.DataChannel) {
		p.DataChannel = d
		d.OnMessage(func(msg webrtc.DataChannelMessage) {
			if msg.IsString {
				if p.OnMessage != nil {
					p.OnMessage(string(msg.Data))
				}
			} else {
				if p.OnBinary != nil {
					p.OnBinary(msg.Data)
				}
			}
		})
		if p.OnDataChannel != nil {
			p.OnDataChannel(d)
		}
	}

	if isHost {
		pc.OnDataChannel(func(d *webrtc.DataChannel) {
			log.Println("Data channel established!")
			setupDataChannel(d)
		})
	} else {
		d, err := pc.CreateDataChannel("control", nil)
		if err != nil {
			return nil, err
		}
		d.OnOpen(func() {
			log.Println("Data channel opened!")
			d.SendText("Hello from Remote!")
		})
		setupDataChannel(d)
	}

	return p, nil
}

func (p *Peer) HandleSignal(signal string) error {
	var candidate webrtc.ICECandidateInit
	if err := json.Unmarshal([]byte(signal), &candidate); err == nil && candidate.Candidate != "" {
		return p.Connection.AddICECandidate(candidate)
	}

	var desc webrtc.SessionDescription
	if err := json.Unmarshal([]byte(signal), &desc); err == nil {
		if err := p.Connection.SetRemoteDescription(desc); err != nil {
			return err
		}
		if desc.Type == webrtc.SDPTypeOffer {
			answer, err := p.Connection.CreateAnswer(nil)
			if err != nil {
				return err
			}
			if err := p.Connection.SetLocalDescription(answer); err != nil {
				return err
			}
			b, _ := json.Marshal(answer)
			p.OnSignal(string(b))
		}
	}
	return nil
}

func (p *Peer) CreateOffer() error {
	offer, err := p.Connection.CreateOffer(nil)
	if err != nil {
		return err
	}
	if err := p.Connection.SetLocalDescription(offer); err != nil {
		return err
	}
	b, _ := json.Marshal(offer)
	p.OnSignal(string(b))
	return nil
}
