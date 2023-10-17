package peer

import (
	"net/http"

	"github.com/olahol/melody"
)

type Server struct {
	melody *melody.Melody
}

func (s *Server) HandleRequest(w http.ResponseWriter, r *http.Request) {
	s.melody.HandleRequestWithKeys(w, r, map[string]interface{}{})
}

func New() *Server {
	m := melody.New()

	s := &Server{
		melody: m,
	}

	m.HandleConnect(func(s *melody.Session) {
		relay := NewMelodyRelay(s)
		session := NewSession(relay)
		s.Keys["relay"] = relay
		s.Keys["session"] = session
	})

	m.HandleMessage(func(s *melody.Session, msg []byte) {
		getRelay(s).internalHandleData(msg)
	})

	m.HandleDisconnect(func(s *melody.Session) {
		getRelay(s).Close()
	})

	return s
}

func getSession(s *melody.Session) *Session {
	return s.Keys["session"].(*Session)
}

func getRelay(s *melody.Session) *MelodyRelay {
	return s.Keys["relay"].(*MelodyRelay)
}

type MelodyRelay struct {
	melodySession *melody.Session
	dataCallback  func(data string)
	closeCallback func()
}

func (m *MelodyRelay) Close() {
	if m.closeCallback != nil {
		m.closeCallback()
	}
	m.melodySession.Close()
}

func (m *MelodyRelay) Send(data string) error {
	return m.melodySession.Write([]byte(data))
}

func (m *MelodyRelay) OnData(callback func(data string)) {
	m.dataCallback = callback
}

func (m *MelodyRelay) OnClose(callback func()) {
	m.closeCallback = callback
}

func (m *MelodyRelay) internalHandleData(data []byte) {
	if m.dataCallback != nil {
		m.dataCallback(string(data))
	}
}

var _ Relay = (*MelodyRelay)(nil)

func NewMelodyRelay(m *melody.Session) *MelodyRelay {
	return &MelodyRelay{
		melodySession: m,
	}
}
