package tplink

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net"
	"net/textproto"
	"sbipc/pkg/mtsp"
	"sync"

	"github.com/pion/rtp"
)

type Conn struct {
	tcp       net.Conn
	conn      *mtsp.Conn
	seq       int
	writeLock *sync.Mutex
}

func (c *Conn) Handshake(username, password string) error {
	c.writeLock.Lock()
	defer c.writeLock.Unlock()

	headers := textproto.MIMEHeader{}
	headers.Add("Authorization", fmt.Sprintf("Basic %s", base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("%s:%s", username, password)))))
	headers.Add("X-Handshake", "unused debug")

	c.conn.WriteMultiTrans(&headers, []byte{})

	r, err := c.conn.Read()
	if err != nil {
		return fmt.Errorf("conn write: %w", err)
	}
	if r.StatusCode != 200 {
		return fmt.Errorf("status %d: %s", r.StatusCode, r.Status)
	}

	return nil
}

type talkResult struct {
	Type   string `json:"type"`
	Seq    int    `json:"seq"`
	Params struct {
		ErrorCode int    `json:"error_code"`
		SessionID string `json:"session_id"`
	} `json:"params"`
}

func (c *Conn) StartTalk() (string, error) {
	c.writeLock.Lock()
	defer c.writeLock.Unlock()

	headers := textproto.MIMEHeader{}
	headers.Add("Content-Type", "application/json")

	c.conn.WriteMultiTrans(&headers, []byte(fmt.Sprintf(`{"type":"request","seq":%d,"params":{"method":"get","talk":{"mode":"aec"}}}`, c.seq)))

	r, err := c.conn.Read()
	if err != nil {
		return "", fmt.Errorf("conn write: %w", err)
	}
	if r.StatusCode != 200 {
		return "", fmt.Errorf("status %d: %s", r.StatusCode, r.Status)
	}

	c.seq++

	var resp talkResult
	if err = json.Unmarshal(r.Body, &resp); err != nil {
		return "", fmt.Errorf("unmarshal: %w", err)
	}

	return resp.Params.SessionID, nil
}

func (c *Conn) WriteTalk(rtpBody []byte) error {
	c.writeLock.Lock()
	defer c.writeLock.Unlock()

	rtpHeader := rtp.Header{
		Version:        2,
		PayloadType:    65,
		SequenceNumber: 0,
		Timestamp:      0x15f90,
		SSRC:           0x78,
	}
	rtpHeaderBytes, _ := rtpHeader.Marshal()

	packetBody := make([]byte, len(rtpBody)+len(rtpHeaderBytes))
	copy(packetBody, rtpHeaderBytes)
	copy(packetBody[len(rtpHeaderBytes):], rtpBody)

	c.conn.WriteInterleaved(packetBody)

	return nil
}

func (c *Conn) StopTalk(sessionId string) error {
	c.writeLock.Lock()
	defer c.writeLock.Unlock()

	headers := textproto.MIMEHeader{}
	headers.Add("Content-Type", "application/json")
	headers.Add("X-Session-Id", sessionId)

	c.conn.WriteMultiTrans(&headers, []byte(fmt.Sprintf(`{"type":"request","seq":%d,"params":{"method":"do","stop":"null"}}`, c.seq)))

	r, err := c.conn.Read()
	if err != nil {
		return fmt.Errorf("conn write: %w", err)
	}
	if r.StatusCode != 200 {
		return fmt.Errorf("status %d: %s", r.StatusCode, r.Status)
	}

	c.seq++

	return nil
}

func (c *Conn) Close() {
	c.writeLock.Lock()
	defer c.writeLock.Unlock()

	c.conn.WriteTeardown()
	c.tcp.Close()
}

func Dial(address string) (*Conn, error) {
	tcp, err := net.Dial("tcp", address)
	if err != nil {
		return nil, err
	}

	conn := &Conn{
		tcp:       tcp,
		conn:      mtsp.NewConn(tcp),
		writeLock: &sync.Mutex{},
	}

	return conn, nil
}
