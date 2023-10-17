package peer

import (
	"log"

	"github.com/hymkor/go-lazy"
	"github.com/pion/interceptor"
	"github.com/pion/webrtc/v4"
)

var webrtcApi = lazy.Of[*webrtc.API]{
	New: func() *webrtc.API {
		mediaEngine := &webrtc.MediaEngine{}
		if err := mediaEngine.RegisterDefaultCodecs(); err != nil {
			log.Fatalf("failed to register default codecs: %s", err)
		}

		interceptorRegistry := &interceptor.Registry{}
		if err := webrtc.RegisterDefaultInterceptors(mediaEngine, interceptorRegistry); err != nil {
			log.Fatalf("failed to register default interceptors: %s", err)
		}

		api := webrtc.NewAPI(webrtc.WithMediaEngine(mediaEngine), webrtc.WithInterceptorRegistry(interceptorRegistry))

		return api
	},
}
