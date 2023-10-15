package mtsp

import (
	"bufio"
	"encoding/binary"
	"fmt"
	"io"
	"net/textproto"
	"regexp"
	"strconv"
	"strings"
	"sync"
)

type Conn struct {
	underlying io.ReadWriteCloser
	reader     *textproto.Reader
	cseq       int
	writeLock  *sync.Mutex
}

type Packet struct {
	IsInterleaved bool
	Status        string
	StatusCode    int
	Headers       *textproto.MIMEHeader
	Channel       int
	Body          []byte
}

func (c *Conn) WriteMultiTrans(headers *textproto.MIMEHeader, body []byte) error {
	return c.WriteText("MULTITRANS", headers, body)
}

func (c *Conn) WriteTeardown() error {
	return c.WriteText("TEARDOWN", &textproto.MIMEHeader{}, []byte{})
}

func (c *Conn) WriteText(method string, headers *textproto.MIMEHeader, body []byte) error {
	c.writeLock.Lock()
	defer c.writeLock.Unlock()

	buf := make([]string, 0)
	buf = append(buf, fmt.Sprintf("%s rtsp://127.0.0.1/multitrans RTSP/1.0", method))
	buf = append(buf, fmt.Sprintf("CSeq: %d", c.cseq))
	buf = append(buf, fmt.Sprintf("Content-Length: %d", len(body)))

	for key, values := range *headers {
		for _, v := range values {
			buf = append(buf, fmt.Sprintf("%s: %s", key, v))
		}
	}

	header := []byte(strings.Join(buf, "\r\n") + "\r\n\r\n")

	payload := make([]byte, len(header)+len(body))
	copy(payload, header)
	copy(payload[len(header):], body)

	if _, err := c.underlying.Write(payload); err != nil {
		return fmt.Errorf("write: %w", err)
	}

	c.cseq++

	return nil
}

func (c *Conn) WriteInterleaved(data []byte) error {
	c.writeLock.Lock()
	defer c.writeLock.Unlock()

	channel := 0
	payload := make([]byte, len(data)+4)
	payload[0] = '$'
	payload[1] = byte(channel)
	binary.BigEndian.PutUint16(payload[2:], uint16(len(data)))
	copy(payload[4:], data)

	c.underlying.Write(payload)
	return nil
}

func (c *Conn) Read() (*Packet, error) {
	b, err := c.reader.R.Peek(1)
	if err != nil {
		return nil, fmt.Errorf("peek: %w", err)
	}

	if b[0] == '$' {
		// binary
		if _, err = c.reader.R.Discard(1); err != nil {
			return nil, fmt.Errorf("read rtsp header: %w", err)
		}

		channel, err := c.reader.R.ReadByte()
		if err != nil {
			return nil, fmt.Errorf("read rtsp channel: %w", err)
		}

		var rtspInterleavedFrameLength uint16
		if err = binary.Read(c.reader.R, binary.BigEndian, &rtspInterleavedFrameLength); err != nil {
			return nil, fmt.Errorf("read rtsp interleaved frame length: %w", err)
		}

		rtspInterleavedFrame := make([]byte, rtspInterleavedFrameLength)
		if _, err = io.ReadFull(c.reader.R, rtspInterleavedFrame); err != nil {
			return nil, fmt.Errorf("read rtsp interleaved frame: %w", err)
		}

		p := &Packet{
			IsInterleaved: true,
			Headers:       nil,
			Body:          rtspInterleavedFrame,
			Channel:       int(channel),
		}

		return p, nil
	} else {
		// text
		status, err := c.reader.ReadLine()
		if err != nil {
			return nil, fmt.Errorf("read status line: %w", err)
		}

		re := regexp.MustCompile(`(?m)RTSP/1\.0\s(\d+)`)
		m := re.FindStringSubmatch(status)
		statusCode, _ := strconv.Atoi(m[1])

		headers, err := c.reader.ReadMIMEHeader()
		if err != nil {
			return nil, fmt.Errorf("read mime header: %w", err)
		}

		var body []byte
		lenS := headers.Get("Content-Length")
		if lenS != "" {
			lenI, err := strconv.Atoi(lenS)
			if err != nil {
				return nil, fmt.Errorf("parse Content-Length: %w", err)
			}

			body = make([]byte, lenI)
			_, err = io.ReadFull(c.reader.R, body)
			if err != nil {
				return nil, fmt.Errorf("read body: %w", err)
			}
		}

		p := &Packet{
			IsInterleaved: false,
			Status:        status,
			StatusCode:    statusCode,
			Headers:       &headers,
			Body:          body,
		}

		return p, nil
	}
}

func NewConn(underlying io.ReadWriteCloser) *Conn {
	return &Conn{
		underlying: underlying,
		reader:     textproto.NewReader(bufio.NewReader(underlying)),
		cseq:       0,
		writeLock:  &sync.Mutex{},
	}
}
