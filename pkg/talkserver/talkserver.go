package talkserver

import (
	"log"
	"net/http"
	"sbipc/pkg/tplink"

	"github.com/olahol/melody"
)

type Server struct {
	melody *melody.Melody
}

type Session struct {
	tpConn    *tplink.Conn
	sessionId string
}

func (s *Server) HandleRequest(w http.ResponseWriter, r *http.Request) {
	log.Printf("Handling websocket request from %s", r.RemoteAddr)

	address := r.URL.Query().Get("address")
	username := r.URL.Query().Get("username")
	password := r.URL.Query().Get("password")

	conn, err := tplink.Dial(address)
	if err != nil {
		log.Printf("dial tplink error: %s", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if err := conn.Handshake(username, password); err != nil {
		log.Printf("handshake tplink error: %s", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	sessionId, err := conn.StartTalk()
	if err != nil {
		log.Printf("start talk error: %s", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	session := &Session{
		tpConn:    conn,
		sessionId: sessionId,
	}

	if err = s.melody.HandleRequestWithKeys(w, r, map[string]interface{}{"session": session}); err != nil {
		log.Printf("upgrade error: %s", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func New() *Server {
	m := melody.New()

	m.HandleMessageBinary(func(s *melody.Session, msg []byte) {
		if err := s.Keys["session"].(*Session).tpConn.WriteTalk(msg); err != nil {
			log.Printf("write error: %s", err)
			s.CloseWithMsg([]byte("internal error"))
		}
	})

	m.HandleDisconnect(func(s *melody.Session) {
		session := s.Keys["session"].(*Session)
		session.tpConn.StopTalk(session.sessionId)
		session.tpConn.Close()
	})

	s := &Server{
		melody: m,
	}

	return s
}
