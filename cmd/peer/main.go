package main

import (
	"net/http"
	"sbipc/pkg/peer"
)

func main() {
	peerServer := peer.NewServer()

	http.HandleFunc("/ipc", func(w http.ResponseWriter, r *http.Request) {
		peerServer.HandleRequest(w, r)
	})

	http.ListenAndServe(":8957", nil)
}
