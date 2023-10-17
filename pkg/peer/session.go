package peer

import (
	"encoding/json"
	"fmt"
	"sbipc/pkg/tplink"
	"sync"

	"github.com/pion/webrtc/v4"
)

type Session struct {
	tpConnPreview    *tplink.Conn
	tpConnTalk       *tplink.Conn
	tpPreviewSession string
	tpTalkSession    string
	peerConnection   *webrtc.PeerConnection
	relay            Relay
	enableTalk       bool
	audioTrack       *webrtc.TrackLocalStaticRTP
	videoTrack       *webrtc.TrackLocalStaticRTP
	talkReceiver     *webrtc.RTPTransceiver
	processLock      *sync.Mutex
}

func (s *Session) onRelayData(data string) {
	s.processLock.Lock()
	defer s.processLock.Unlock()

	var relayData RelayData
	if err := json.Unmarshal([]byte(data), &relayData); err != nil {
		errRelayData := RelayData{
			Success: wrapBool(false),
			Error: &RelayError{
				Message: err.Error(),
			},
		}
		text, _ := json.Marshal(errRelayData)
		s.relay.Send(string(text))
	}

	if err := s.processRelayData(&relayData); err != nil {
		errRelayData := RelayData{
			UserData: relayData.UserData,
			Success:  wrapBool(false),
			Error: &RelayError{
				Message: err.Error(),
			},
		}
		text, _ := json.Marshal(errRelayData)
		s.relay.Send(string(text))
	} else {
		errRelayData := RelayData{
			UserData: relayData.UserData,
			Success:  wrapBool(true),
		}
		text, _ := json.Marshal(errRelayData)
		s.relay.Send(string(text))
	}
}

func (s *Session) processRelayData(relayData *RelayData) error {
	if relayData.Open != nil {
		return s.open(relayData)
	}

	if s.tpConnPreview == nil || s.peerConnection == nil {
		return fmt.Errorf("not open")
	}

	if relayData.SessionDescription != nil {
		if err := s.peerConnection.SetRemoteDescription(*relayData.SessionDescription); err != nil {
			return fmt.Errorf("set remote description: %w", err)
		}
	}

	if relayData.Candidate != nil {
		if err := s.peerConnection.AddICECandidate(*relayData.Candidate); err != nil {
			return fmt.Errorf("add ice candidate: %w", err)
		}
	}

	return fmt.Errorf("invalid request")
}

func (s *Session) open(relayData *RelayData) error {
	if s.tpConnPreview != nil {
		return fmt.Errorf("already open")
	}

	c, err := tplink.Dial(relayData.Open.Address)

	if err != nil {
		return fmt.Errorf("dial: %w", err)
	}

	if err := c.Handshake(relayData.Open.Username, relayData.Open.Password); err != nil {
		return fmt.Errorf("handshake: %w", err)
	}
	s.tpConnPreview = c

	s.enableTalk = relayData.Open.EnableTalk
	if s.enableTalk {
		c, err := tplink.Dial(relayData.Open.Address)

		if err != nil {
			return fmt.Errorf("dial: %w", err)
		}

		if err := c.Handshake(relayData.Open.Username, relayData.Open.Password); err != nil {
			return fmt.Errorf("handshake: %w", err)
		}
		s.tpConnTalk = c
	}

	peerConnection, err := webrtcApi.Value().NewPeerConnection(webrtc.Configuration{
		ICEServers: []webrtc.ICEServer{{URLs: []string{"stun:stun.l.google.com:19302"}}},
	})
	if err != nil {
		return fmt.Errorf("failed to create peer connection: %w", err)
	}
	s.peerConnection = peerConnection

	videoTrack, err := webrtc.NewTrackLocalStaticRTP(webrtc.RTPCodecCapability{MimeType: webrtc.MimeTypeH264}, "video", "preview")
	if err != nil {
		return fmt.Errorf("failed to create video track: %w", err)
	}
	s.videoTrack = videoTrack

	audioTrack, err := webrtc.NewTrackLocalStaticRTP(webrtc.RTPCodecCapability{MimeType: webrtc.MimeTypePCMA}, "audio", "preview")
	if err != nil {
		return fmt.Errorf("failed to create audio track: %w", err)
	}
	s.audioTrack = audioTrack

	_, err = peerConnection.AddTrack(videoTrack)
	if err != nil {
		return fmt.Errorf("failed to add video track: %w", err)
	}

	_, err = peerConnection.AddTrack(audioTrack)
	if err != nil {
		return fmt.Errorf("failed to add audio track: %w", err)
	}

	if s.enableTalk {
		talkReceiver, err := peerConnection.AddTransceiverFromKind(webrtc.RTPCodecTypeAudio)
		if err != nil {
			return fmt.Errorf("failed to add transceiver: %w", err)
		}
		s.talkReceiver = talkReceiver
	}

	peerConnection.OnICECandidate(func(candidate *webrtc.ICECandidate) {
		candidateInit := candidate.ToJSON()
		relayData := &RelayData{
			Candidate: &candidateInit,
		}
		text, _ := json.Marshal(relayData)
		s.relay.Send(string(text))
	})

	peerConnection.OnICEConnectionStateChange(func(connectionState webrtc.ICEConnectionState) {
		if connectionState == webrtc.ICEConnectionStateFailed {
			s.relay.Close()
		}
	})

	sd, err := peerConnection.CreateOffer(nil)
	if err != nil {
		return fmt.Errorf("failed to create offer: %w", err)
	}
	if err := peerConnection.SetLocalDescription(sd); err != nil {
		return fmt.Errorf("failed to set local description: %w", err)
	}

	offerData := &RelayData{
		SessionDescription: &sd,
	}
	offerDataText, _ := json.Marshal(offerData)
	s.relay.Send(string(offerDataText))

	return nil
}

func (s *Session) onClose() {
	if s.peerConnection != nil {
		s.peerConnection.Close()
	}
	if s.tpConnTalk != nil {
		s.tpConnTalk.StopTalk(s.tpTalkSession)
		s.tpConnTalk.Close()
	}
	if s.tpConnPreview != nil {
		s.tpConnPreview.StopTalk(s.tpPreviewSession)
		s.tpConnPreview.Close()
	}
}

func NewSession(relay Relay) *Session {
	s := &Session{
		relay:       relay,
		processLock: &sync.Mutex{},
	}

	relay.OnData(s.onRelayData)
	relay.OnClose(s.onClose)

	return s
}
