package peer

import (
	"github.com/pion/webrtc/v4"
)

type Relay interface {
	Send(data string) error
	OnData(callback func(data string))
	OnClose(callback func())
	Close()
}

type RelayError struct {
	Message string `json:"message"`
}

type RelayData struct {
	UserData           string                     `json:"userData"`
	SessionDescription *webrtc.SessionDescription `json:"sessionDescription"`
	Candidate          *webrtc.ICECandidateInit   `json:"candidate"`
	Open               *struct {
		Address    string `json:"address"`
		Username   string `json:"username"`
		Password   string `json:"password"`
		EnableTalk bool   `json:"enableTalk"`
	} `json:"open"`
	Error   *RelayError `json:"error"`
	Success *bool       `json:"success"`
}

func wrapBool(v bool) *bool {
	return &v
}
