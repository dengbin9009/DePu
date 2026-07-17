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

	"github.com/dengbin9009/DePu/backend/internal/game"
	"github.com/dengbin9009/DePu/backend/internal/storage"
)

const websocketGUID = "258EAFA5-E914-47DA-95CA-C5AB0DC85B11"

type socketEnvelope struct {
	Type        string          `json:"type"`
	RequestID   string          `json:"requestId,omitempty"`
	RoomID      string          `json:"roomId,omitempty"`
	Payload     json.RawMessage `json:"payload,omitempty"`
	SentAt      string          `json:"sentAt,omitempty"`
	RoomVersion int64           `json:"roomVersion,omitempty"`
	HandID      string          `json:"handId,omitempty"`
	HandVersion int             `json:"handVersion,omitempty"`
}

type socketHub struct {
	mu             sync.RWMutex
	clients        map[*socketClient]struct{}
	rooms          map[string]map[*socketClient]struct{}
	roomMu         map[string]*sync.Mutex
	presence       map[string]map[string]*roomPresence
	actionLogs     map[string][]roomActionLogEntry
	chatMessages   map[string][]roomChatMessage
	nextChatByUser map[string]time.Time
	timerVersion   map[string]int
}

type socketClient struct {
	server *Server
	userID string
	conn   net.Conn

	sendMu sync.Mutex
	rooms  map[string]struct{}
}

type roomPresence struct {
	UserID             string  `json:"userId"`
	SeatNo             int     `json:"seatNo,omitempty"`
	Status             string  `json:"status"`
	LastDisconnectedAt *string `json:"lastDisconnectedAt"`
	connections        int
}

type roomActionLogEntry struct {
	HandID    string `json:"handId,omitempty"`
	Seq       int    `json:"seq"`
	Kind      string `json:"kind"`
	Street    string `json:"street,omitempty"`
	SeatNo    int    `json:"seatNo,omitempty"`
	Nickname  string `json:"nickname,omitempty"`
	Action    string `json:"action,omitempty"`
	Amount    int    `json:"amount,omitempty"`
	Source    string `json:"source"`
	CreatedAt string `json:"createdAt"`
}

type roomChatMessage struct {
	ID        string `json:"id"`
	Kind      string `json:"kind"`
	Text      string `json:"text,omitempty"`
	EmojiCode string `json:"emojiCode,omitempty"`
	UserID    string `json:"userId"`
	Nickname  string `json:"nickname"`
	CreatedAt string `json:"createdAt"`
}

func newSocketHub() *socketHub {
	return &socketHub{
		clients:        map[*socketClient]struct{}{},
		rooms:          map[string]map[*socketClient]struct{}{},
		roomMu:         map[string]*sync.Mutex{},
		presence:       map[string]map[string]*roomPresence{},
		actionLogs:     map[string][]roomActionLogEntry{},
		chatMessages:   map[string][]roomChatMessage{},
		nextChatByUser: map[string]time.Time{},
		timerVersion:   map[string]int{},
	}
}

func (h *socketHub) add(client *socketClient) {
	h.mu.Lock()
	h.clients[client] = struct{}{}
	h.mu.Unlock()
}

func (h *socketHub) subscribe(client *socketClient, roomID string) bool {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.rooms[roomID] == nil {
		h.rooms[roomID] = map[*socketClient]struct{}{}
	}
	if _, subscribed := h.rooms[roomID][client]; subscribed {
		return false
	}
	h.rooms[roomID][client] = struct{}{}
	client.rooms[roomID] = struct{}{}
	return true
}

func (h *socketHub) unsubscribe(client *socketClient, roomID string) {
	h.mu.Lock()
	if roomClients, ok := h.rooms[roomID]; ok {
		delete(roomClients, client)
		if len(roomClients) == 0 {
			delete(h.rooms, roomID)
		}
	}
	delete(client.rooms, roomID)
	h.mu.Unlock()
}

func (h *socketHub) unsubscribeUser(roomID, userID string) {
	h.mu.Lock()
	if roomClients, ok := h.rooms[roomID]; ok {
		for client := range roomClients {
			if client.userID != userID {
				continue
			}
			delete(roomClients, client)
			delete(client.rooms, roomID)
		}
		if len(roomClients) == 0 {
			delete(h.rooms, roomID)
		}
	}
	if roomPresence := h.presence[roomID]; roomPresence != nil {
		delete(roomPresence, userID)
		if len(roomPresence) == 0 {
			delete(h.presence, roomID)
		}
	}
	h.mu.Unlock()
}

func (h *socketHub) withRoomLock(roomID string, fn func()) {
	h.mu.Lock()
	mu := h.roomMu[roomID]
	if mu == nil {
		mu = &sync.Mutex{}
		h.roomMu[roomID] = mu
	}
	h.mu.Unlock()
	mu.Lock()
	defer mu.Unlock()
	fn()
}

func (h *socketHub) roomClients(roomID string) []*socketClient {
	h.mu.RLock()
	defer h.mu.RUnlock()
	clients := make([]*socketClient, 0, len(h.rooms[roomID]))
	for client := range h.rooms[roomID] {
		clients = append(clients, client)
	}
	return clients
}

func (h *socketHub) remove(client *socketClient) {
	h.mu.Lock()
	changed := make([]roomPresence, 0)
	delete(h.clients, client)
	for roomID := range client.rooms {
		if roomClients, ok := h.rooms[roomID]; ok {
			delete(roomClients, client)
			if len(roomClients) == 0 {
				delete(h.rooms, roomID)
			}
		}
		if presence := h.presence[roomID][client.userID]; presence != nil {
			presence.connections--
			if presence.connections <= 0 {
				presence.connections = 0
				presence.Status = "offline"
				now := time.Now().UTC().Format(time.RFC3339Nano)
				presence.LastDisconnectedAt = &now
				changed = append(changed, *presence)
			}
		}
	}
	h.mu.Unlock()
	for _, item := range changed {
		client.server.broadcastRoomPresence(item)
	}
}

func (h *socketHub) markOnline(room *storage.RoomRecord, userID string) (roomPresence, bool) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.presence[room.ID] == nil {
		h.presence[room.ID] = map[string]*roomPresence{}
	}
	seatNo := 0
	for _, seat := range room.Seats {
		if seat.UserID != nil && *seat.UserID == userID {
			seatNo = seat.SeatNo
			break
		}
	}
	item := h.presence[room.ID][userID]
	isFirstPresence := item == nil
	wasOnline := item != nil && item.connections > 0
	if item == nil {
		item = &roomPresence{UserID: userID}
		h.presence[room.ID][userID] = item
	}
	wasOffline := item.connections == 0 && item.Status == "offline"
	item.SeatNo = seatNo
	item.Status = "online"
	item.LastDisconnectedAt = nil
	item.connections++
	return *item, isFirstPresence || (wasOffline && !wasOnline)
}

func (h *socketHub) presenceSnapshot(room *storage.RoomRecord) []roomPresence {
	h.mu.RLock()
	defer h.mu.RUnlock()
	items := make([]roomPresence, 0, len(room.Members))
	for _, member := range room.Members {
		seatNo := 0
		for _, seat := range room.Seats {
			if seat.UserID != nil && *seat.UserID == member.UserID {
				seatNo = seat.SeatNo
				break
			}
		}
		if existing := h.presence[room.ID][member.UserID]; existing != nil {
			copy := *existing
			copy.SeatNo = seatNo
			items = append(items, copy)
			continue
		}
		items = append(items, roomPresence{UserID: member.UserID, SeatNo: seatNo, Status: "offline"})
	}
	return items
}

func (h *socketHub) appendActionLog(roomID string, entry roomActionLogEntry) roomActionLogEntry {
	h.mu.Lock()
	defer h.mu.Unlock()
	entry.Seq = len(h.actionLogs[roomID]) + 1
	entry.CreatedAt = time.Now().UTC().Format(time.RFC3339Nano)
	h.actionLogs[roomID] = append(h.actionLogs[roomID], entry)
	if len(h.actionLogs[roomID]) > 50 {
		h.actionLogs[roomID] = h.actionLogs[roomID][len(h.actionLogs[roomID])-50:]
	}
	return entry
}

func (h *socketHub) recentActionLog(roomID string) []roomActionLogEntry {
	h.mu.RLock()
	defer h.mu.RUnlock()
	items := append([]roomActionLogEntry{}, h.actionLogs[roomID]...)
	return items
}

func (h *socketHub) appendChat(roomID string, message roomChatMessage) roomChatMessage {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.chatMessages[roomID] = append(h.chatMessages[roomID], message)
	if len(h.chatMessages[roomID]) > 30 {
		h.chatMessages[roomID] = h.chatMessages[roomID][len(h.chatMessages[roomID])-30:]
	}
	return message
}

func (h *socketHub) recentChat(roomID string) []roomChatMessage {
	h.mu.RLock()
	defer h.mu.RUnlock()
	items := append([]roomChatMessage{}, h.chatMessages[roomID]...)
	return items
}

func (h *socketHub) allowChat(userID string, now time.Time) bool {
	h.mu.Lock()
	defer h.mu.Unlock()
	if next := h.nextChatByUser[userID]; !next.IsZero() && now.Before(next) {
		return false
	}
	h.nextChatByUser[userID] = now.Add(300 * time.Millisecond)
	return true
}

func (h *socketHub) nextTimerVersion(roomID string) int {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.timerVersion[roomID]++
	return h.timerVersion[roomID]
}

func (h *socketHub) currentTimerVersion(roomID string) int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.timerVersion[roomID]
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
	client := &socketClient{server: s, userID: user.ID, conn: conn, rooms: map[string]struct{}{}}
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
		opcode, payload, err := readClientFrame(reader)
		if err != nil || opcode == 0x8 {
			return
		}
		if opcode == 0x1 {
			c.handleMessage(payload)
		}
	}
}

func (c *socketClient) handleMessage(payload []byte) {
	var message socketEnvelope
	if err := json.Unmarshal(payload, &message); err != nil {
		c.writeError(socketEnvelope{}, "bad_message", "invalid socket message", "")
		return
	}
	switch message.Type {
	case "room.subscribe":
		c.handleSubscribe(message)
	case "room.unsubscribe":
		c.server.hub.unsubscribe(c, message.RoomID)
		c.writeAck(message)
	case "room.start_hand":
		c.handleStartHand(message)
	case "room.action":
		c.handleAction(message)
	case "room.leave":
		c.handleLeaveRoom(message)
	case "chat.send":
		c.handleChatSend(message)
	default:
		c.writeError(message, "bad_message", "unsupported socket message type", "type")
	}
}

func (c *socketClient) handleSubscribe(message socketEnvelope) {
	room, err := c.server.store.RoomByID(message.RoomID)
	if err != nil {
		c.writeError(message, "room_not_found", "room not found", "roomId")
		return
	}
	if !roomHasMember(room, c.userID) {
		c.writeError(message, "forbidden", "room subscription requires membership", "roomId")
		return
	}
	newSubscription := c.server.hub.subscribe(c, message.RoomID)
	var presence roomPresence
	changed := false
	if newSubscription {
		presence, changed = c.server.hub.markOnline(room, c.userID)
	}
	c.writeAck(message)
	c.writeRoomSnapshot(message, room)
	if changed {
		c.server.broadcastRoomPresence(presence)
	}
}

func (c *socketClient) handleStartHand(message socketEnvelope) {
	c.server.hub.withRoomLock(message.RoomID, func() {
		result, apiErr := c.server.startRoomHandForUser(message.RoomID, c.userID)
		if apiErr != nil {
			c.writeAPIError(message, apiErr)
			return
		}
		c.writeAck(message)
		c.server.broadcastHandState(message.RoomID, "hand.started", message.RequestID, result.Game, int64FromMap(result.State, "roomVersion"))
		entry := c.server.hub.appendActionLog(message.RoomID, roomActionLogEntry{
			HandID: stringFromMap(result.State, "handId"),
			Kind:   "hand_started",
			Street: stringFromMap(result.State, "status"),
			SeatNo: intFromMap(result.State, "currentSeat"),
			Source: "system",
		})
		c.server.broadcastRoomEvent(message.RoomID, socketEnvelope{
			Type:      "hand.log.appended",
			RequestID: message.RequestID,
			RoomID:    message.RoomID,
			Payload:   mustJSON(map[string]any{"entry": entry}),
			SentAt:    time.Now().UTC().Format(time.RFC3339),
		})
	})
}

func (c *socketClient) handleLeaveRoom(message socketEnvelope) {
	c.server.hub.withRoomLock(message.RoomID, func() {
		room, apiErr := c.server.leaveRoomForUser(message.RoomID, c.userID)
		if apiErr != nil {
			c.writeAPIError(message, apiErr)
			return
		}
		c.writeAck(message)
		c.server.hub.unsubscribeUser(message.RoomID, c.userID)
		c.server.broadcastRoomUpdate(message.RoomID, room)
		c.server.broadcastRoomLog(message.RoomID, "room_left", c.userID, 0)
	})
}

func (c *socketClient) handleAction(message socketEnvelope) {
	var req struct {
		Action string `json:"action"`
		Amount int    `json:"amount"`
	}
	if err := json.Unmarshal(message.Payload, &req); err != nil {
		c.writeError(message, "bad_message", "invalid action payload", "payload")
		return
	}
	c.server.hub.withRoomLock(message.RoomID, func() {
		result, apiErr := c.server.applyRoomActionForUser(message.RoomID, c.userID, req.Action, req.Amount)
		if apiErr != nil {
			c.writeAPIError(message, apiErr)
			return
		}
		c.writeAck(message)
		if result.Settled && result.Result != nil {
			c.server.broadcastRoomEvent(message.RoomID, socketEnvelope{
				Type:        "hand.settled",
				RequestID:   message.RequestID,
				RoomID:      message.RoomID,
				Payload:     mustJSON(map[string]any{"hand": publicHandResult(result.Result)}),
				SentAt:      time.Now().UTC().Format(time.RFC3339),
				RoomVersion: result.RoomVersion,
				HandID:      result.HandID,
				HandVersion: result.HandVersion,
			})
			c.server.sendWalletUpdates(message.RoomID, message.RequestID, result.Result.Participants, result.Result.HandID)
			c.server.broadcastLeaderboard(message.RoomID, message.RequestID)
			if room, err := c.server.store.RoomByID(message.RoomID); err == nil {
				c.server.broadcastRoomUpdate(message.RoomID, room)
				c.server.scheduleAutoNextHand(message.RoomID)
			}
			return
		}
		c.server.broadcastHandState(message.RoomID, "hand.updated", message.RequestID, result.Game, result.RoomVersion)
		entry := c.server.hub.appendActionLog(message.RoomID, roomActionLogEntry{
			HandID: stringFromMap(result.State, "handId"),
			Kind:   "player_action",
			Street: stringFromMap(result.State, "status"),
			SeatNo: intFromMap(result.State, "currentSeat"),
			Action: req.Action,
			Amount: req.Amount,
			Source: "player",
		})
		c.server.broadcastRoomEvent(message.RoomID, socketEnvelope{
			Type:      "hand.log.appended",
			RequestID: message.RequestID,
			RoomID:    message.RoomID,
			Payload:   mustJSON(map[string]any{"entry": entry}),
			SentAt:    time.Now().UTC().Format(time.RFC3339),
		})
	})
}

func (c *socketClient) handleChatSend(message socketEnvelope) {
	room, err := c.server.store.RoomByID(message.RoomID)
	if err != nil {
		c.writeError(message, "room_not_found", "room not found", "roomId")
		return
	}
	if !roomHasMember(room, c.userID) {
		c.writeError(message, "forbidden", "chat requires room membership", "roomId")
		return
	}
	var req struct {
		Kind      string `json:"kind"`
		Text      string `json:"text"`
		EmojiCode string `json:"emojiCode"`
	}
	if err := json.Unmarshal(message.Payload, &req); err != nil {
		c.writeError(message, "bad_message", "invalid chat payload", "payload")
		return
	}
	req.Kind = strings.TrimSpace(req.Kind)
	req.Text = strings.TrimSpace(req.Text)
	req.EmojiCode = strings.TrimSpace(req.EmojiCode)
	switch req.Kind {
	case "text":
		if req.Text == "" {
			c.writeError(message, "chat_message_empty", "chat message cannot be empty", "text")
			return
		}
		if len([]rune(req.Text)) > 200 {
			c.writeError(message, "chat_message_too_long", "chat message is too long", "text")
			return
		}
	case "emoji":
		if !allowedEmojiCode(req.EmojiCode) {
			c.writeError(message, "chat_emoji_unknown", "unknown emoji", "emojiCode")
			return
		}
	default:
		c.writeError(message, "bad_message", "unsupported chat kind", "kind")
		return
	}
	if !c.server.hub.allowChat(c.userID, time.Now().UTC()) {
		c.writeError(message, "chat_rate_limited", "chat messages are too frequent", "")
		return
	}
	user, err := c.server.store.FindUserByID(c.userID)
	if err != nil {
		c.writeError(message, "unauthorized", "user not found", "")
		return
	}
	now := time.Now().UTC().Format(time.RFC3339Nano)
	chat := c.server.hub.appendChat(message.RoomID, roomChatMessage{
		ID:        "chat_" + itoa(time.Now().UTC().Nanosecond()),
		Kind:      req.Kind,
		Text:      req.Text,
		EmojiCode: req.EmojiCode,
		UserID:    c.userID,
		Nickname:  user.Nickname,
		CreatedAt: now,
	})
	c.writeAck(message)
	c.server.broadcastRoomEvent(message.RoomID, socketEnvelope{
		Type:      "chat.message",
		RequestID: message.RequestID,
		RoomID:    message.RoomID,
		Payload:   mustJSON(map[string]any{"message": chat}),
		SentAt:    time.Now().UTC().Format(time.RFC3339),
	})
}

func (c *socketClient) writeAck(request socketEnvelope) {
	payload, _ := json.Marshal(map[string]string{"command": request.Type})
	_ = c.writeJSON(socketEnvelope{
		Type:      "ack",
		RequestID: request.RequestID,
		RoomID:    request.RoomID,
		Payload:   payload,
		SentAt:    time.Now().UTC().Format(time.RFC3339),
	})
}

func (c *socketClient) writeError(request socketEnvelope, code, message, field string) {
	payload, _ := json.Marshal(ErrorResponse{Code: code, Message: message, Field: field})
	_ = c.writeJSON(socketEnvelope{
		Type:      "error",
		RequestID: request.RequestID,
		RoomID:    request.RoomID,
		Payload:   payload,
		SentAt:    time.Now().UTC().Format(time.RFC3339),
	})
}

func (c *socketClient) writeAPIError(request socketEnvelope, apiErr *apiError) {
	c.writeError(request, apiErr.Code, apiErr.Message, apiErr.Field)
}

func (c *socketClient) writeRoomSnapshot(request socketEnvelope, room *storage.RoomRecord) {
	var hand any
	var handID string
	var handVersion int
	if room.CurrentGameID != "" {
		if g, err := c.server.store.Load(room.CurrentGameID); err == nil {
			hand = c.server.roomHandStateForUser(room, g, c.userID)
			handID = g.ID
			handVersion = g.Version
		}
	}
	leaderboard, _ := c.server.store.RoomLeaderboard(room.ID, 10)
	payload, _ := json.Marshal(map[string]any{
		"room":               room,
		"hand":               hand,
		"presence":           c.server.hub.presenceSnapshot(room),
		"recentActionLog":    c.server.hub.recentActionLog(room.ID),
		"recentChatMessages": c.server.hub.recentChat(room.ID),
		"leaderboard":        leaderboard,
	})
	_ = c.writeJSON(socketEnvelope{
		Type:        "room.snapshot",
		RequestID:   request.RequestID,
		RoomID:      request.RoomID,
		Payload:     payload,
		SentAt:      time.Now().UTC().Format(time.RFC3339),
		RoomVersion: room.Version,
		HandID:      handID,
		HandVersion: handVersion,
	})
}

func allowedEmojiCode(code string) bool {
	switch code {
	case "nice_hand", "good_luck", "wow", "thanks", "all_in":
		return true
	default:
		return false
	}
}

func roomHasMember(room *storage.RoomRecord, userID string) bool {
	for _, member := range room.Members {
		if member.UserID == userID {
			return true
		}
	}
	return false
}

func (s *Server) broadcastRoomEvent(roomID string, event socketEnvelope) {
	for _, client := range s.hub.roomClients(roomID) {
		_ = client.writeJSON(event)
	}
}

func (s *Server) broadcastHandState(roomID, eventType, requestID string, g *game.Game, roomVersion int64) {
	if g == nil {
		return
	}
	room, err := s.store.RoomByID(roomID)
	if err != nil {
		return
	}
	for _, client := range s.hub.roomClients(roomID) {
		state := s.roomHandStateForUser(room, g, client.userID)
		state["roomVersion"] = roomVersion
		_ = client.writeJSON(socketEnvelope{
			Type:        eventType,
			RequestID:   requestID,
			RoomID:      roomID,
			Payload:     mustJSON(map[string]any{"hand": state}),
			SentAt:      time.Now().UTC().Format(time.RFC3339),
			RoomVersion: roomVersion,
			HandID:      g.ID,
			HandVersion: g.Version,
		})
	}
}

func (s *Server) sendWalletUpdates(roomID, requestID string, participants []storage.HandParticipantRecord, handID string) {
	userIDs := map[string]struct{}{}
	for _, participant := range participants {
		userIDs[participant.UserID] = struct{}{}
	}
	for _, client := range s.hub.roomClients(roomID) {
		if _, ok := userIDs[client.userID]; !ok {
			continue
		}
		_ = client.writeJSON(socketEnvelope{
			Type:      "wallet.updated",
			RequestID: requestID,
			RoomID:    roomID,
			Payload: mustJSON(map[string]string{
				"reason": "hand_settled",
				"handId": handID,
			}),
			SentAt: time.Now().UTC().Format(time.RFC3339),
		})
	}
}

func (s *Server) broadcastRoomUpdate(roomID string, room *storage.RoomRecord) {
	s.broadcastRoomEvent(roomID, socketEnvelope{
		Type:        "room.updated",
		RoomID:      roomID,
		Payload:     mustJSON(map[string]any{"room": room}),
		SentAt:      time.Now().UTC().Format(time.RFC3339),
		RoomVersion: room.Version,
	})
}

func (s *Server) broadcastRoomLog(roomID, kind, userID string, seatNo int) {
	nickname := ""
	if user, err := s.store.FindUserByID(userID); err == nil {
		nickname = user.Nickname
	}
	entry := s.hub.appendActionLog(roomID, roomActionLogEntry{
		Kind:     kind,
		SeatNo:   seatNo,
		Nickname: nickname,
		Source:   "system",
	})
	s.broadcastRoomEvent(roomID, socketEnvelope{
		Type:    "hand.log.appended",
		RoomID:  roomID,
		Payload: mustJSON(map[string]any{"entry": entry}),
		SentAt:  time.Now().UTC().Format(time.RFC3339),
	})
}

func (s *Server) broadcastRoomPresence(item roomPresence) {
	roomID := ""
	s.hub.mu.RLock()
	for candidateRoomID, members := range s.hub.presence {
		if members[item.UserID] != nil {
			roomID = candidateRoomID
			break
		}
	}
	s.hub.mu.RUnlock()
	if roomID == "" {
		return
	}
	s.broadcastRoomEvent(roomID, socketEnvelope{
		Type:    "player.presence.updated",
		RoomID:  roomID,
		Payload: mustJSON(item),
		SentAt:  time.Now().UTC().Format(time.RFC3339),
	})
}

func (s *Server) broadcastLeaderboard(roomID, requestID string) {
	items, err := s.store.RoomLeaderboard(roomID, 10)
	if err != nil {
		return
	}
	s.broadcastRoomEvent(roomID, socketEnvelope{
		Type:      "room.leaderboard.updated",
		RequestID: requestID,
		RoomID:    roomID,
		Payload:   mustJSON(map[string]any{"items": items}),
		SentAt:    time.Now().UTC().Format(time.RFC3339),
	})
}

func (s *Server) scheduleActionTimeout(roomID, handID string, seatNo int) {
	if seatNo == 0 || handID == "" {
		return
	}
	timeout := s.actionTimeout
	if timeout <= 0 {
		timeout = 30 * time.Second
	}
	version := s.hub.nextTimerVersion(roomID)
	go func() {
		time.Sleep(timeout)
		if s.hub.currentTimerVersion(roomID) != version {
			return
		}
		s.applyTimeoutAction(roomID, handID, seatNo, version)
	}()
}

func (s *Server) applyTimeoutAction(roomID, handID string, seatNo, version int) {
	s.hub.withRoomLock(roomID, func() {
		if s.hub.currentTimerVersion(roomID) != version {
			return
		}
		room, err := s.store.RoomByID(roomID)
		if err != nil || room.CurrentGameID != handID {
			return
		}
		g, err := s.store.Load(handID)
		if err != nil || g.CurrentSeat != seatNo || g.Stage == game.StageFinished {
			return
		}
		action := string(game.ActionFold)
		for _, candidate := range g.LegalActions() {
			if candidate == game.ActionCheck {
				action = string(game.ActionCheck)
				break
			}
		}
		result, apiErr := s.applyRoomActionForSeat(roomID, g, action, 0, "")
		if apiErr != nil {
			return
		}
		s.broadcastRoomEvent(roomID, socketEnvelope{
			Type:   "hand.timeout_applied",
			RoomID: roomID,
			Payload: mustJSON(map[string]any{
				"handId":    handID,
				"seatNo":    seatNo,
				"action":    action,
				"source":    "timeout",
				"appliedAt": time.Now().UTC().Format(time.RFC3339Nano),
			}),
			SentAt: time.Now().UTC().Format(time.RFC3339),
		})
		entry := s.hub.appendActionLog(roomID, roomActionLogEntry{
			HandID: handID,
			Kind:   "timeout_action",
			Street: stringFromMap(result.State, "status"),
			SeatNo: seatNo,
			Action: action,
			Source: "timeout",
		})
		s.broadcastRoomEvent(roomID, socketEnvelope{
			Type:    "hand.log.appended",
			RoomID:  roomID,
			Payload: mustJSON(map[string]any{"entry": entry}),
			SentAt:  time.Now().UTC().Format(time.RFC3339),
		})
		if result.Settled && result.Result != nil {
			s.broadcastRoomEvent(roomID, socketEnvelope{
				Type:        "hand.settled",
				RoomID:      roomID,
				Payload:     mustJSON(map[string]any{"hand": publicHandResult(result.Result)}),
				SentAt:      time.Now().UTC().Format(time.RFC3339),
				RoomVersion: result.RoomVersion,
				HandID:      result.HandID,
				HandVersion: result.HandVersion,
			})
			s.sendWalletUpdates(roomID, "", result.Result.Participants, result.Result.HandID)
			s.broadcastLeaderboard(roomID, "")
			if room, roomErr := s.store.RoomByID(roomID); roomErr == nil {
				s.broadcastRoomUpdate(roomID, room)
				s.scheduleAutoNextHand(roomID)
			}
			return
		}
		s.broadcastHandState(roomID, "hand.updated", "", result.Game, result.RoomVersion)
	})
}

func stringFromMap(value map[string]any, key string) string {
	if text, ok := value[key].(string); ok {
		return text
	}
	return ""
}

func intFromMap(value map[string]any, key string) int {
	switch raw := value[key].(type) {
	case int:
		return raw
	case float64:
		return int(raw)
	default:
		return 0
	}
}

func int64FromMap(value map[string]any, key string) int64 {
	switch raw := value[key].(type) {
	case int64:
		return raw
	case int:
		return int64(raw)
	case float64:
		return int64(raw)
	default:
		return 0
	}
}

func mustJSON(value any) json.RawMessage {
	payload, _ := json.Marshal(value)
	return payload
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
