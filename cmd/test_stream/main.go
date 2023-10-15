package main

import (
	"log"
	"os"
	"sbipc/pkg/tplink"

	"github.com/pion/rtp"
	"github.com/pion/webrtc/v4/pkg/media/h264writer"
)

func main() {
	conn, err := tplink.Dial("192.168.20.2:554")
	if err != nil {
		log.Fatal(err)
	}

	if err := conn.Handshake("admin", os.Getenv("IPC_PASSWORD")); err != nil {
		log.Fatal(err)
	}

	preview, err := conn.StartPreview()
	if err != nil {
		log.Fatal(err)
	}

	log.Printf("started preview: %#v", preview)

	w, err := h264writer.New("test.h264")
	if err != nil {
		log.Fatal(err)
	}

	for {
		p, err := conn.Read()
		if err != nil {
			log.Fatal(err)
		}
		if p.IsInterleaved {
			// log.Printf("frame channel: %d", p.Channel)
			var rp rtp.Packet
			err = rp.Unmarshal(p.Body)
			if err != nil {
				log.Fatal(err)
			}

			if p.Channel == 0 {
				w.WriteRTP(&rp)
			}
		}
	}
}
