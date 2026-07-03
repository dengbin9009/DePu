package api

import (
	"bufio"
	"crypto/sha1"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"errors"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/dengbin9009/DePu/backend/internal/storage"
)

const websocketGUID = "258EAFA5-E914-47DA-95CA-C5AB0DC85B11"

type socketEnvelope struct {
	Type      string          `json:"type"`
	RequestID string          `json:"requestId,omitempty"`
	RoomID    string          `json:"roomId,omitempty"`
	Payload   json.RawMessage `json:"payload,omitempty"`
	SentAt    string          `json:"sentAt,omitempty"`
}

type socketHub struct {
	mu      sync.RWMutex
	clients map[*socketClient]struct{}
	rooms   map[string]map[*socketClient]struct{}
}

type socketClient struct {
	userID string
	conn   net.Conn

	sendMu sync.Mutex
	rooms  map[string]struct{}
}

func newSocketHub() *socketHub {
	return &socketHub{
		clients: map[*socketClient]struct{}{},
		rooms:   map[string]map[*socketClient]struct{}{},
	}
}

func (h *socketHub) add(client *socketClient) {
	h.mu.Lock()
	h.clients[client] = struct{}{}
	h.mu.Unlock()
}

func (h *socketHub) remove(client *socketClient) {
	h.mu.Lock()
	delete(h.clients, client)
	for roomID := range client.rooms {
		if roomClients, ok := h.rooms[roomID]; ok {
			delete(roomClients, client)
			if len(roomClients) == 0 {
				delete(h.rooms, roomID)
			}
		}
	}
	h.mu.Unlock()
}

func (s *Server) socketEndpoint(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed", "")
		return
	}
	user, err := s.socketUser(r)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "unauthorized", "authentication required", "")
		return
	}
	conn, err := upgradeWebSocket(w, r)
	if err != nil {
		writeError(w, http.StatusBadRequest, "bad_websocket", err.Error(), "")
		return
	}
	client := &socketClient{userID: user.ID, conn: conn, rooms: map[string]struct{}{}}
	s.hub.add(client)
	go func() {
		defer conn.Close()
		defer s.hub.remove(client)
		readyPayload, _ := json.Marshal(map[string]string{
			"userId":          user.ID,
			"protocolVersion": "1",
			"serverTime":      time.Now().UTC().Format(time.RFC3339),
		})
		_ = client.writeJSON(socketEnvelope{
			Type:    "connection.ready",
			Payload: readyPayload,
			SentAt:  time.Now().UTC().Format(time.RFC3339),
		})
		client.readLoop()
	}()
}

func (s *Server) socketUser(r *http.Request) (*storage.UserRecord, error) {
	token := strings.TrimSpace(r.URL.Query().Get("token"))
	if token == "" {
		auth := strings.TrimSpace(r.Header.Get("Authorization"))
		if strings.HasPrefix(auth, "Bearer ") {
			token = strings.TrimSpace(strings.TrimPrefix(auth, "Bearer "))
		}
	}
	if token == "" {
		return nil, errors.New("missing token")
	}
	s.mu.RLock()
	userID, ok := s.sessions[token]
	s.mu.RUnlock()
	if !ok {
		return nil, errors.New("invalid session")
	}
	return s.store.FindUserByID(userID)
}

func upgradeWebSocket(w http.ResponseWriter, r *http.Request) (net.Conn, error) {
	if !headerHasToken(r.Header, "Connection", "upgrade") || !strings.EqualFold(r.Header.Get("Upgrade"), "websocket") {
		return nil, errors.New("missing websocket upgrade headers")
	}
	key := strings.TrimSpace(r.Header.Get("Sec-WebSocket-Key"))
	if key == "" || r.Header.Get("Sec-WebSocket-Version") != "13" {
		return nil, errors.New("invalid websocket handshake")
	}
	hijacker, ok := w.(http.Hijacker)
	if !ok {
		return nil, errors.New("websocket hijacking unavailable")
	}
	conn, rw, err := hijacker.Hijack()
	if err != nil {
		return nil, err
	}
	accept := websocketAccept(key)
	_, err = rw.WriteString("HTTP/1.1 101 Switching Protocols\r\n" +
		"Upgrade: websocket\r\n" +
		"Connection: Upgrade\r\n" +
		"Sec-WebSocket-Accept: " + accept + "\r\n" +
		"\r\n")
	if err != nil {
		conn.Close()
		return nil, err
	}
	if err := rw.Flush(); err != nil {
		conn.Close()
		return nil, err
	}
	return conn, nil
}

func websocketAccept(key string) string {
	sum := sha1.Sum([]byte(key + websocketGUID))
	return base64.StdEncoding.EncodeToString(sum[:])
}

func headerHasToken(header http.Header, name, token string) bool {
	for _, value := range header.Values(name) {
		for _, part := range strings.Split(value, ",") {
			if strings.EqualFold(strings.TrimSpace(part), token) {
				return true
			}
		}
	}
	return false
}

func (c *socketClient) writeJSON(message socketEnvelope) error {
	payload, err := json.Marshal(message)
	if err != nil {
		return err
	}
	return c.writeText(payload)
}

func (c *socketClient) writeText(payload []byte) error {
	c.sendMu.Lock()
	defer c.sendMu.Unlock()

	header := []byte{0x81}
	switch {
	case len(payload) < 126:
		header = append(header, byte(len(payload)))
	case len(payload) <= 65535:
		header = append(header, 126, byte(len(payload)>>8), byte(len(payload)))
	default:
		extended := make([]byte, 8)
		binary.BigEndian.PutUint64(extended, uint64(len(payload)))
		header = append(header, 127)
		header = append(header, extended...)
	}
	if _, err := c.conn.Write(header); err != nil {
		return err
	}
	_, err := c.conn.Write(payload)
	return err
}

func (c *socketClient) readLoop() {
	reader := bufio.NewReader(c.conn)
	for {
		opcode, _, err := readClientFrame(reader)
		if err != nil || opcode == 0x8 {
			return
		}
	}
}

func readClientFrame(reader *bufio.Reader) (byte, []byte, error) {
	header := make([]byte, 2)
	if _, err := reader.Read(header); err != nil {
		return 0, nil, err
	}
	opcode := header[0] & 0x0f
	masked := header[1]&0x80 != 0
	length := uint64(header[1] & 0x7f)
	switch length {
	case 126:
		extended := make([]byte, 2)
		if _, err := reader.Read(extended); err != nil {
			return 0, nil, err
		}
		length = uint64(binary.BigEndian.Uint16(extended))
	case 127:
		extended := make([]byte, 8)
		if _, err := reader.Read(extended); err != nil {
			return 0, nil, err
		}
		length = binary.BigEndian.Uint64(extended)
	}
	var mask []byte
	if masked {
		mask = make([]byte, 4)
		if _, err := reader.Read(mask); err != nil {
			return 0, nil, err
		}
	}
	payload := make([]byte, length)
	if _, err := reader.Read(payload); err != nil {
		return 0, nil, err
	}
	if masked {
		for i := range payload {
			payload[i] ^= mask[i%len(mask)]
		}
	}
	return opcode, payload, nil
}
