package main

import (
	"net/http"
	"sbipc/pkg/talkserver"
)

func main() {
	talkServer := talkserver.New()

	http.HandleFunc("/talk", func(w http.ResponseWriter, r *http.Request) {
		talkServer.HandleRequest(w, r)
	})

	http.Handle("/", http.RedirectHandler("/ui", 302))
	http.Handle("/ui/", http.StripPrefix("/ui", http.FileServer(http.Dir("./ui"))))

	http.ListenAndServe(":8957", nil)
}
