package main

import (
	"flag"
	"log"
	"sbipc/pkg/signal"
	"sbipc/pkg/tplink"

	"github.com/pion/interceptor"
	"github.com/pion/webrtc/v4"
)

func main() {
	var tpAddress string
	var tpUser string
	var tpPass string

	flag.StringVar(&tpAddress, "address", "", "tplink address")
	flag.StringVar(&tpUser, "username", "admin", "tplink username")
	flag.StringVar(&tpPass, "password", "", "tplink password")

	flag.Parse()

	ipc, err := tplink.Dial(tpAddress)
	if err != nil {
		log.Fatalf("failed to dial tplink: %s", err)
	}

	if err := ipc.Handshake(tpUser, tpPass); err != nil {
		log.Fatalf("failed to handshake tplink: %s", err)
	}

	mediaEngine := &webrtc.MediaEngine{}
	if err := mediaEngine.RegisterDefaultCodecs(); err != nil {
		log.Fatalf("failed to register default codecs: %s", err)
	}

	interceptorRegistry := &interceptor.Registry{}
	if err := webrtc.RegisterDefaultInterceptors(mediaEngine, interceptorRegistry); err != nil {
		log.Fatalf("failed to register default interceptors: %s", err)
	}

	api := webrtc.NewAPI(webrtc.WithMediaEngine(mediaEngine), webrtc.WithInterceptorRegistry(interceptorRegistry))

	peerConnection, err := api.NewPeerConnection(webrtc.Configuration{
		ICEServers: []webrtc.ICEServer{{URLs: []string{"stun:stun.l.google.com:19302"}}},
	})
	if err != nil {
		log.Fatalf("failed to create peer connection: %s", err)
	}

	videoTrack, err := webrtc.NewTrackLocalStaticRTP(webrtc.RTPCodecCapability{MimeType: webrtc.MimeTypeH264}, "video", "preview")
	if err != nil {
		log.Fatalf("failed to create video track: %s", err)
	}

	audioTrack, err := webrtc.NewTrackLocalStaticRTP(webrtc.RTPCodecCapability{MimeType: webrtc.MimeTypePCMA}, "audio", "preview")
	if err != nil {
		log.Fatalf("failed to create audio track: %s", err)
	}

	_, err = peerConnection.AddTrack(videoTrack)
	if err != nil {
		log.Fatalf("failed to add video track: %s", err)
	}

	_, err = peerConnection.AddTrack(audioTrack)
	if err != nil {
		log.Fatalf("failed to add audio track: %s", err)
	}

	peerConnection.OnICEConnectionStateChange(func(connectionState webrtc.ICEConnectionState) {
		log.Printf("ICE Connection State has changed: %s\n", connectionState)
		if connectionState == webrtc.ICEConnectionStateFailed {
			log.Println("ICE Connection State failed, closing peer connection")
			if closeErr := peerConnection.Close(); closeErr != nil {
				log.Fatalf("failed to close peer connection: %s\n", closeErr)
			}
		}
	})

	offer := webrtc.SessionDescription{}
	signal.Decode(signal.MustReadStdin(), &offer)
	if err = peerConnection.SetRemoteDescription(offer); err != nil {
		log.Fatalf("failed to set remote description: %s", err)
	}

	answer, err := peerConnection.CreateAnswer(nil)
	if err != nil {
		log.Fatalf("failed to create answer: %s", err)
	}

	gatherComplete := webrtc.GatheringCompletePromise(peerConnection)

	if err = peerConnection.SetLocalDescription(answer); err != nil {
		log.Fatalf("failed to set local description: %s", err)
	}

	<-gatherComplete

	log.Println(signal.Encode(*peerConnection.LocalDescription()))

	_, err = ipc.StartPreview()
	if err != nil {
		log.Fatalf("failed to start preview: %s", err)
	}

	for {
		p, err := ipc.Read()
		if err != nil {
			log.Fatalf("read error: %s", err)
		}

		if p.IsInterleaved {
			if p.Channel == 0 {
				_, err := videoTrack.Write(p.Body)
				if err != nil {
					log.Fatalf("write error: %s", err)
				}
			}

			if p.Channel == 1 {
				_, err := audioTrack.Write(p.Body)
				if err != nil {
					log.Fatalf("write error: %s", err)
				}
			}
		}
	}
}
