package api

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"encoding/json"
	"errors"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/dengbin9009/DePu/backend/internal/game"
)

const socketTestReadTimeout = time.Second

func TestSocketRejectsMissingToken(t *testing.T) {
	server := testServer(t)
	res := httptest.NewRecorder()
	req := socketUpgradeRequest("/api/socket")

	server.Routes().ServeHTTP(res, req)

	if res.Code != http.StatusUnauthorized {
		t.Fatalf("socket status = %d body=%s, want 401", res.Code, res.Body.String())
	}
	var body ErrorResponse
	if err := json.Unmarshal(res.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body.Code != "unauthorized" {
		t.Fatalf("error code = %s, want unauthorized", body.Code)
	}
}

func TestSocketSendsConnectionReadyForValidToken(t *testing.T) {
	server := testServer(t)
	token := registerUser(t, server, "socket_owner", "Socket房主")
	ts := httptest.NewServer(server.Routes())
	defer ts.Close()

	client := dialSocket(t, ts.URL, "/api/socket?token="+token)
	defer client.Close()

	readUpgradeResponse(t, client)
	_, payload := readServerTextFrame(t, client)
	var msg struct {
		Type    string          `json:"type"`
		Payload json.RawMessage `json:"payload"`
	}
	if err := json.Unmarshal(payload, &msg); err != nil {
		t.Fatal(err)
	}
	if msg.Type != "connection.ready" {
		t.Fatalf("message type = %s, want connection.ready; payload=%s", msg.Type, string(payload))
	}
	var ready struct {
		UserID          string `json:"userId"`
		ProtocolVersion string `json:"protocolVersion"`
		ServerTime      string `json:"serverTime"`
	}
	if err := json.Unmarshal(msg.Payload, &ready); err != nil {
		t.Fatal(err)
	}
	if ready.UserID == "" || ready.ProtocolVersion == "" || ready.ServerTime == "" {
		t.Fatalf("ready payload incomplete: %#v", ready)
	}
}

func TestSocketKeepsConnectionOpenAfterReady(t *testing.T) {
	server := testServer(t)
	token := registerUser(t, server, "socket_open", "Socket长连")
	ts := httptest.NewServer(server.Routes())
	defer ts.Close()

	client := dialSocket(t, ts.URL, "/api/socket?token="+token)
	defer client.Close()
	readUpgradeResponse(t, client)
	readServerTextFrame(t, client)

	if err := writeClientTextFrame(client, []byte(`{"type":"room.refresh","requestId":"req_keepalive"}`)); err != nil {
		t.Fatal(err)
	}
	if err := writeClientCloseFrame(client); err != nil {
		t.Fatal(err)
	}
}

func TestSocketHubTracksActiveConnectionUntilClientCloses(t *testing.T) {
	server := testServer(t)
	token := registerUser(t, server, "socket_tracked", "Socket追踪")
	ts := httptest.NewServer(server.Routes())
	defer ts.Close()

	client := dialSocket(t, ts.URL, "/api/socket?token="+token)
	defer client.Close()
	readUpgradeResponse(t, client)
	readServerTextFrame(t, client)
	time.Sleep(20 * time.Millisecond)

	if got := socketHubClientCount(server.hub); got != 1 {
		t.Fatalf("active socket clients = %d, want 1", got)
	}
	if err := writeClientCloseFrame(client); err != nil {
		t.Fatal(err)
	}
	eventually(t, func() bool { return socketHubClientCount(server.hub) == 0 })
}

func TestSocketSubscribeReturnsRoomSnapshotForMember(t *testing.T) {
	server := testServer(t)
	roomID, ownerToken, _ := setupRoomWithSeats(t, server)
	ts := httptest.NewServer(server.Routes())
	defer ts.Close()

	client := dialSocket(t, ts.URL, "/api/socket?token="+ownerToken)
	defer client.Close()
	readUpgradeResponse(t, client)
	readSocketMessage(t, client, "connection.ready")

	if err := writeClientTextFrame(client, []byte(`{"type":"room.subscribe","requestId":"req_subscribe","roomId":"`+roomID+`","payload":{}}`)); err != nil {
		t.Fatal(err)
	}
	ack := readSocketMessage(t, client, "ack")
	if ack.RequestID != "req_subscribe" || ack.RoomID != roomID {
		t.Fatalf("ack = %#v, want request and room ids", ack)
	}
	snapshot := readSocketMessage(t, client, "room.snapshot")
	if snapshot.RoomID != roomID {
		t.Fatalf("snapshot roomId = %s, want %s", snapshot.RoomID, roomID)
	}
	var payload struct {
		Room struct {
			ID      string `json:"id"`
			Members []struct {
				UserID string `json:"userId"`
			} `json:"members"`
			Seats []struct {
				SeatNo int `json:"seatNo"`
			} `json:"seats"`
		} `json:"room"`
		Hand any `json:"hand"`
	}
	if err := json.Unmarshal(snapshot.Payload, &payload); err != nil {
		t.Fatal(err)
	}
	if payload.Room.ID != roomID {
		t.Fatalf("payload room id = %s, want %s", payload.Room.ID, roomID)
	}
	if len(payload.Room.Members) != 2 {
		t.Fatalf("members = %d, want 2", len(payload.Room.Members))
	}
	if len(payload.Room.Seats) != 6 {
		t.Fatalf("seats = %d, want 6", len(payload.Room.Seats))
	}
	if got := socketHubRoomClientCount(server.hub, roomID); got != 1 {
		t.Fatalf("subscribed room clients = %d, want 1", got)
	}
}

func TestSocketSubscribeSnapshotIncludesCurrentHand(t *testing.T) {
	server := testServer(t)
	roomID, ownerToken, _ := setupRoomWithSeats(t, server)
	startReq := httptest.NewRequest(http.MethodPost, "/api/rooms/"+roomID+"/start", nil)
	startReq.Header.Set("Authorization", "Bearer "+ownerToken)
	startRes := httptest.NewRecorder()
	server.Routes().ServeHTTP(startRes, startReq)
	if startRes.Code != http.StatusOK {
		t.Fatalf("start status=%d body=%s", startRes.Code, startRes.Body.String())
	}
	ts := httptest.NewServer(server.Routes())
	defer ts.Close()

	client := dialSocket(t, ts.URL, "/api/socket?token="+ownerToken)
	defer client.Close()
	readUpgradeResponse(t, client)
	readSocketMessage(t, client, "connection.ready")

	if err := writeClientTextFrame(client, []byte(`{"type":"room.subscribe","requestId":"req_subscribe_hand","roomId":"`+roomID+`","payload":{}}`)); err != nil {
		t.Fatal(err)
	}
	readSocketMessage(t, client, "ack")
	snapshot := readSocketMessage(t, client, "room.snapshot")
	var payload struct {
		Hand *struct {
			RoomID      string   `json:"roomId"`
			HandID      string   `json:"handId"`
			CurrentSeat int      `json:"currentSeat"`
			Players     []any    `json:"players"`
			Available   []string `json:"availableActions"`
			BoardCards  []string `json:"boardCards"`
			Status      string   `json:"status"`
		} `json:"hand"`
	}
	if err := json.Unmarshal(snapshot.Payload, &payload); err != nil {
		t.Fatal(err)
	}
	if payload.Hand == nil {
		t.Fatal("expected current hand snapshot")
	}
	if payload.Hand.RoomID != roomID || payload.Hand.HandID == "" || payload.Hand.CurrentSeat == 0 {
		t.Fatalf("hand snapshot incomplete: %#v", payload.Hand)
	}
	if len(payload.Hand.Players) == 0 || len(payload.Hand.Available) == 0 {
		t.Fatalf("hand players/actions missing: %#v", payload.Hand)
	}
}

func TestSocketSubscribeRejectsNonMember(t *testing.T) {
	server := testServer(t)
	roomID, _, _ := setupRoomWithSeats(t, server)
	outsiderToken := registerUser(t, server, "socket_outsider", "Socket旁观")
	ts := httptest.NewServer(server.Routes())
	defer ts.Close()

	client := dialSocket(t, ts.URL, "/api/socket?token="+outsiderToken)
	defer client.Close()
	readUpgradeResponse(t, client)
	readSocketMessage(t, client, "connection.ready")

	if err := writeClientTextFrame(client, []byte(`{"type":"room.subscribe","requestId":"req_forbidden","roomId":"`+roomID+`","payload":{}}`)); err != nil {
		t.Fatal(err)
	}
	msg := readSocketMessage(t, client, "error")
	if msg.RequestID != "req_forbidden" || msg.RoomID != roomID {
		t.Fatalf("error envelope = %#v, want request and room ids", msg)
	}
	var payload ErrorResponse
	if err := json.Unmarshal(msg.Payload, &payload); err != nil {
		t.Fatal(err)
	}
	if payload.Code != "forbidden" {
		t.Fatalf("error code = %s, want forbidden", payload.Code)
	}
	if got := socketHubRoomClientCount(server.hub, roomID); got != 0 {
		t.Fatalf("subscribed room clients = %d, want 0", got)
	}
}

func TestSocketUnsubscribeRemovesRoomSubscription(t *testing.T) {
	server := testServer(t)
	roomID, ownerToken, _ := setupRoomWithSeats(t, server)
	ts := httptest.NewServer(server.Routes())
	defer ts.Close()

	client := dialSocket(t, ts.URL, "/api/socket?token="+ownerToken)
	defer client.Close()
	readUpgradeResponse(t, client)
	readSocketMessage(t, client, "connection.ready")

	if err := writeClientTextFrame(client, []byte(`{"type":"room.subscribe","requestId":"req_subscribe","roomId":"`+roomID+`","payload":{}}`)); err != nil {
		t.Fatal(err)
	}
	readSocketMessage(t, client, "ack")
	readSocketMessage(t, client, "room.snapshot")

	if err := writeClientTextFrame(client, []byte(`{"type":"room.unsubscribe","requestId":"req_unsubscribe","roomId":"`+roomID+`","payload":{}}`)); err != nil {
		t.Fatal(err)
	}
	ack := readSocketMessage(t, client, "ack")
	if ack.RequestID != "req_unsubscribe" {
		t.Fatalf("unsubscribe ack requestId = %s, want req_unsubscribe", ack.RequestID)
	}
	if got := socketHubRoomClientCount(server.hub, roomID); got != 0 {
		t.Fatalf("subscribed room clients = %d, want 0", got)
	}
}

func TestSocketOwnerCanStartHandAndSubscribersReceiveStarted(t *testing.T) {
	server := testServer(t)
	roomID, ownerToken, playerToken := setupRoomWithSeats(t, server)
	ts := httptest.NewServer(server.Routes())
	defer ts.Close()
	ownerClient := dialSocket(t, ts.URL, "/api/socket?token="+ownerToken)
	defer ownerClient.Close()
	playerClient := dialSocket(t, ts.URL, "/api/socket?token="+playerToken)
	defer playerClient.Close()
	subscribeSocket(t, ownerClient, roomID, "req_owner_sub")
	subscribeSocket(t, playerClient, roomID, "req_player_sub")

	if err := writeClientTextFrame(ownerClient, []byte(`{"type":"room.start_hand","requestId":"req_start","roomId":"`+roomID+`","payload":{}}`)); err != nil {
		t.Fatal(err)
	}
	ack := readSocketMessage(t, ownerClient, "ack")
	if ack.RequestID != "req_start" {
		t.Fatalf("start ack requestId = %s, want req_start", ack.RequestID)
	}
	ownerStarted := readSocketMessage(t, ownerClient, "hand.started")
	playerStarted := readSocketMessage(t, playerClient, "hand.started")
	for _, msg := range []socketEnvelope{ownerStarted, playerStarted} {
		if msg.RoomID != roomID {
			t.Fatalf("started roomId = %s, want %s", msg.RoomID, roomID)
		}
		var payload struct {
			Hand struct {
				RoomID      string `json:"roomId"`
				HandID      string `json:"handId"`
				CurrentSeat int    `json:"currentSeat"`
			} `json:"hand"`
		}
		if err := json.Unmarshal(msg.Payload, &payload); err != nil {
			t.Fatal(err)
		}
		if payload.Hand.RoomID != roomID || payload.Hand.HandID == "" || payload.Hand.CurrentSeat == 0 {
			t.Fatalf("hand.started payload incomplete: %#v", payload.Hand)
		}
	}
}

func TestSocketNonOwnerCannotStartHand(t *testing.T) {
	server := testServer(t)
	roomID, _, playerToken := setupRoomWithSeats(t, server)
	ts := httptest.NewServer(server.Routes())
	defer ts.Close()
	playerClient := dialSocket(t, ts.URL, "/api/socket?token="+playerToken)
	defer playerClient.Close()
	subscribeSocket(t, playerClient, roomID, "req_player_sub")

	if err := writeClientTextFrame(playerClient, []byte(`{"type":"room.start_hand","requestId":"req_bad_start","roomId":"`+roomID+`","payload":{}}`)); err != nil {
		t.Fatal(err)
	}
	msg := readSocketMessage(t, playerClient, "error")
	var payload ErrorResponse
	if err := json.Unmarshal(msg.Payload, &payload); err != nil {
		t.Fatal(err)
	}
	if payload.Code != "not_room_owner" {
		t.Fatalf("error code = %s, want not_room_owner", payload.Code)
	}
}

func TestSocketCurrentPlayerCanActAndSubscribersReceiveUpdate(t *testing.T) {
	server := testServer(t)
	roomID, ownerToken, playerToken := setupRoomWithSeats(t, server)
	ts := httptest.NewServer(server.Routes())
	defer ts.Close()
	ownerClient := dialSocket(t, ts.URL, "/api/socket?token="+ownerToken)
	defer ownerClient.Close()
	playerClient := dialSocket(t, ts.URL, "/api/socket?token="+playerToken)
	defer playerClient.Close()
	subscribeSocket(t, ownerClient, roomID, "req_owner_sub")
	subscribeSocket(t, playerClient, roomID, "req_player_sub")
	startHandViaSocket(t, ownerClient, playerClient, roomID)

	current := currentHandViaHTTP(t, server, roomID, ownerToken)
	actorClient := ownerClient
	observerClient := playerClient
	if current.CurrentSeat == 2 {
		actorClient = playerClient
		observerClient = ownerClient
	}
	action := "call"
	if hasAction(current.AvailableActions, "check") {
		action = "check"
	}
	if err := writeClientTextFrame(actorClient, []byte(`{"type":"room.action","requestId":"req_action","roomId":"`+roomID+`","payload":{"action":"`+action+`","amount":0}}`)); err != nil {
		t.Fatal(err)
	}
	ack := readSocketMessage(t, actorClient, "ack")
	if ack.RequestID != "req_action" {
		t.Fatalf("action ack requestId = %s, want req_action", ack.RequestID)
	}
	actorUpdate := readSocketMessage(t, actorClient, "hand.updated")
	observerUpdate := readSocketMessage(t, observerClient, "hand.updated")
	for _, msg := range []socketEnvelope{actorUpdate, observerUpdate} {
		var payload struct {
			Hand struct {
				HandID      string `json:"handId"`
				CurrentSeat int    `json:"currentSeat"`
			} `json:"hand"`
		}
		if err := json.Unmarshal(msg.Payload, &payload); err != nil {
			t.Fatal(err)
		}
		if payload.Hand.HandID == "" || payload.Hand.CurrentSeat == current.CurrentSeat {
			t.Fatalf("hand.updated did not advance action: before=%d payload=%#v", current.CurrentSeat, payload.Hand)
		}
	}
}

func TestSocketNonCurrentPlayerCannotAct(t *testing.T) {
	server := testServer(t)
	roomID, ownerToken, playerToken := setupRoomWithSeats(t, server)
	ts := httptest.NewServer(server.Routes())
	defer ts.Close()
	ownerClient := dialSocket(t, ts.URL, "/api/socket?token="+ownerToken)
	defer ownerClient.Close()
	playerClient := dialSocket(t, ts.URL, "/api/socket?token="+playerToken)
	defer playerClient.Close()
	subscribeSocket(t, ownerClient, roomID, "req_owner_sub")
	subscribeSocket(t, playerClient, roomID, "req_player_sub")
	startHandViaSocket(t, ownerClient, playerClient, roomID)

	current := currentHandViaHTTP(t, server, roomID, ownerToken)
	nonActorClient := playerClient
	if current.CurrentSeat == 2 {
		nonActorClient = ownerClient
	}
	if err := writeClientTextFrame(nonActorClient, []byte(`{"type":"room.action","requestId":"req_bad_action","roomId":"`+roomID+`","payload":{"action":"call","amount":0}}`)); err != nil {
		t.Fatal(err)
	}
	msg := readSocketMessage(t, nonActorClient, "error")
	var payload ErrorResponse
	if err := json.Unmarshal(msg.Payload, &payload); err != nil {
		t.Fatal(err)
	}
	if payload.Code != "not_your_turn" {
		t.Fatalf("error code = %s, want not_your_turn", payload.Code)
	}
	after := currentHandViaHTTP(t, server, roomID, ownerToken)
	if after.CurrentSeat != current.CurrentSeat || after.HandID != current.HandID {
		t.Fatalf("non-current action changed hand: before=%#v after=%#v", current, after)
	}
}

func TestSocketInvalidActionDoesNotMutateHand(t *testing.T) {
	server := testServer(t)
	roomID, ownerToken, playerToken := setupRoomWithSeats(t, server)
	ts := httptest.NewServer(server.Routes())
	defer ts.Close()
	ownerClient := dialSocket(t, ts.URL, "/api/socket?token="+ownerToken)
	defer ownerClient.Close()
	playerClient := dialSocket(t, ts.URL, "/api/socket?token="+playerToken)
	defer playerClient.Close()
	subscribeSocket(t, ownerClient, roomID, "req_owner_sub")
	subscribeSocket(t, playerClient, roomID, "req_player_sub")
	startHandViaSocket(t, ownerClient, playerClient, roomID)

	current := currentHandViaHTTP(t, server, roomID, ownerToken)
	actorClient := ownerClient
	if current.CurrentSeat == 2 {
		actorClient = playerClient
	}
	if err := writeClientTextFrame(actorClient, []byte(`{"type":"room.action","requestId":"req_invalid_action","roomId":"`+roomID+`","payload":{"action":"raise","amount":1}}`)); err != nil {
		t.Fatal(err)
	}
	msg := readSocketMessage(t, actorClient, "error")
	var payload ErrorResponse
	if err := json.Unmarshal(msg.Payload, &payload); err != nil {
		t.Fatal(err)
	}
	if payload.Code != "invalid_action" {
		t.Fatalf("error code = %s, want invalid_action", payload.Code)
	}
	after := currentHandViaHTTP(t, server, roomID, ownerToken)
	if after.CurrentSeat != current.CurrentSeat || after.HandID != current.HandID {
		t.Fatalf("invalid action changed hand: before=%#v after=%#v", current, after)
	}
}

func TestSocketSettledHandBroadcastsSettlementAndWalletUpdates(t *testing.T) {
	server := testServer(t)
	roomID, ownerToken, playerToken := setupRoomWithSeats(t, server)
	ts := httptest.NewServer(server.Routes())
	defer ts.Close()
	ownerClient := dialSocket(t, ts.URL, "/api/socket?token="+ownerToken)
	defer ownerClient.Close()
	playerClient := dialSocket(t, ts.URL, "/api/socket?token="+playerToken)
	defer playerClient.Close()
	subscribeSocket(t, ownerClient, roomID, "req_owner_sub")
	subscribeSocket(t, playerClient, roomID, "req_player_sub")
	startHandViaSocket(t, ownerClient, playerClient, roomID)

	var settled socketEnvelope
	var ownerWalletUpdates int
	var playerWalletUpdates int
	for i := 0; i < 12; i++ {
		current, ok := tryCurrentHandViaHTTP(t, server, roomID, ownerToken)
		if !ok {
			break
		}
		if current.Status == "finished" {
			break
		}
		action := preferredAction(current.AvailableActions)
		actor := ownerClient
		observer := playerClient
		if current.CurrentSeat == 2 {
			actor = playerClient
			observer = ownerClient
		}
		if err := writeClientTextFrame(actor, []byte(`{"type":"room.action","requestId":"req_settle_`+itoa(i)+`","roomId":"`+roomID+`","payload":{"action":"`+action+`","amount":0}}`)); err != nil {
			t.Fatal(err)
		}
		readSocketMessage(t, actor, "ack")
		first := readSocketMessageAny(t, actor)
		second := readSocketMessageAny(t, observer)
		for _, msg := range []socketEnvelope{first, second} {
			if msg.Type == "hand.settled" {
				settled = msg
			}
			if msg.Type == "wallet.updated" {
				if actor == ownerClient {
					ownerWalletUpdates++
				} else {
					playerWalletUpdates++
				}
			}
		}
		if settled.Type == "hand.settled" {
			ownerWalletUpdates += drainWalletUpdates(t, ownerClient)
			playerWalletUpdates += drainWalletUpdates(t, playerClient)
			break
		}
	}
	if settled.Type != "hand.settled" {
		t.Fatal("expected hand.settled broadcast")
	}
	var payload struct {
		Hand struct {
			HandID        string `json:"handId"`
			WinnerSummary string `json:"winnerSummary"`
			PotSummary    string `json:"potSummary"`
			Participants  []any  `json:"participants"`
			HandNo        int    `json:"handNo"`
			CompletedAt   string `json:"completedAt"`
		} `json:"hand"`
	}
	if err := json.Unmarshal(settled.Payload, &payload); err != nil {
		t.Fatal(err)
	}
	if payload.Hand.HandID == "" || payload.Hand.HandNo == 0 || payload.Hand.WinnerSummary == "" || payload.Hand.PotSummary == "" || len(payload.Hand.Participants) != 2 {
		t.Fatalf("settlement payload incomplete: %#v", payload.Hand)
	}
	if ownerWalletUpdates == 0 || playerWalletUpdates == 0 {
		t.Fatalf("wallet updates owner=%d player=%d, want both > 0", ownerWalletUpdates, playerWalletUpdates)
	}
}

func TestSocketStorageFailureDoesNotBroadcastSuccess(t *testing.T) {
	base := testServer(t)
	roomID, ownerToken, playerToken := setupRoomWithSeats(t, base)
	startReq := httptest.NewRequest(http.MethodPost, "/api/rooms/"+roomID+"/start", nil)
	startReq.Header.Set("Authorization", "Bearer "+ownerToken)
	startRes := httptest.NewRecorder()
	base.Routes().ServeHTTP(startRes, startReq)
	if startRes.Code != http.StatusOK {
		t.Fatalf("start status=%d body=%s", startRes.Code, startRes.Body.String())
	}
	server := NewServerWithStore(&saveFailStore{Store: base.store})
	server.sessions = base.sessions
	ts := httptest.NewServer(server.Routes())
	defer ts.Close()
	ownerClient := dialSocket(t, ts.URL, "/api/socket?token="+ownerToken)
	defer ownerClient.Close()
	playerClient := dialSocket(t, ts.URL, "/api/socket?token="+playerToken)
	defer playerClient.Close()
	subscribeSocket(t, ownerClient, roomID, "req_owner_sub")
	subscribeSocket(t, playerClient, roomID, "req_player_sub")

	current := currentHandViaHTTP(t, base, roomID, ownerToken)
	actorClient := ownerClient
	if current.CurrentSeat == 2 {
		actorClient = playerClient
	}
	if err := writeClientTextFrame(actorClient, []byte(`{"type":"room.action","requestId":"req_storage_fail","roomId":"`+roomID+`","payload":{"action":"call","amount":0}}`)); err != nil {
		t.Fatal(err)
	}
	msg := readSocketMessage(t, actorClient, "error")
	var payload ErrorResponse
	if err := json.Unmarshal(msg.Payload, &payload); err != nil {
		t.Fatal(err)
	}
	if payload.Code != "storage_error" {
		t.Fatalf("error code = %s, want storage_error", payload.Code)
	}
	if msg, ok := tryReadSocketMessage(playerClient, 50*time.Millisecond); ok && (msg.Type == "hand.updated" || msg.Type == "hand.settled") {
		t.Fatalf("unexpected success broadcast after storage failure: %#v", msg)
	}
}

func socketUpgradeRequest(path string) *http.Request {
	req := httptest.NewRequest(http.MethodGet, path, nil)
	req.Header.Set("Connection", "Upgrade")
	req.Header.Set("Upgrade", "websocket")
	req.Header.Set("Sec-WebSocket-Version", "13")
	req.Header.Set("Sec-WebSocket-Key", "dGhlIHNhbXBsZSBub25jZQ==")
	return req
}

type socketTestConn struct {
	net.Conn
	reader *bufio.Reader
}

func dialSocket(t *testing.T, serverURL, path string) *socketTestConn {
	t.Helper()
	addr := strings.TrimPrefix(serverURL, "http://")
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		t.Fatal(err)
	}
	key := "dGhlIHNhbXBsZSBub25jZQ=="
	var req bytes.Buffer
	req.WriteString("GET " + path + " HTTP/1.1\r\n")
	req.WriteString("Host: " + addr + "\r\n")
	req.WriteString("Connection: Upgrade\r\n")
	req.WriteString("Upgrade: websocket\r\n")
	req.WriteString("Sec-WebSocket-Version: 13\r\n")
	req.WriteString("Sec-WebSocket-Key: " + key + "\r\n")
	req.WriteString("\r\n")
	if _, err := conn.Write(req.Bytes()); err != nil {
		conn.Close()
		t.Fatal(err)
	}
	return &socketTestConn{Conn: conn, reader: bufio.NewReader(conn)}
}

func readUpgradeResponse(t *testing.T, client *socketTestConn) {
	t.Helper()
	statusLine, err := client.reader.ReadString('\n')
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(statusLine, "101 Switching Protocols") {
		t.Fatalf("status line = %q, want websocket upgrade", statusLine)
	}
	for {
		line, err := client.reader.ReadString('\n')
		if err != nil {
			t.Fatal(err)
		}
		if line == "\r\n" {
			return
		}
	}
}

func readServerTextFrame(t *testing.T, client *socketTestConn) (byte, []byte) {
	t.Helper()
	_ = client.SetReadDeadline(time.Now().Add(socketTestReadTimeout))
	defer client.SetReadDeadline(time.Time{})
	header := make([]byte, 2)
	if _, err := client.reader.Read(header); err != nil {
		t.Fatal(err)
	}
	opcode := header[0] & 0x0f
	length := int(header[1] & 0x7f)
	if length == 126 {
		extended := make([]byte, 2)
		if _, err := client.reader.Read(extended); err != nil {
			t.Fatal(err)
		}
		length = int(extended[0])<<8 | int(extended[1])
	}
	payload := make([]byte, length)
	if _, err := client.reader.Read(payload); err != nil {
		t.Fatal(err)
	}
	if opcode != 1 {
		t.Fatalf("opcode = %d, want text frame", opcode)
	}
	return opcode, payload
}

func readSocketMessage(t *testing.T, client *socketTestConn, wantType string) socketEnvelope {
	t.Helper()
	_, payload := readServerTextFrame(t, client)
	var msg socketEnvelope
	if err := json.Unmarshal(payload, &msg); err != nil {
		t.Fatal(err)
	}
	if msg.Type != wantType {
		t.Fatalf("message type = %s, want %s; payload=%s", msg.Type, wantType, string(payload))
	}
	return msg
}

func writeClientTextFrame(client *socketTestConn, payload []byte) error {
	return writeClientFrame(client, 0x1, payload)
}

func writeClientCloseFrame(client *socketTestConn) error {
	return writeClientFrame(client, 0x8, nil)
}

func writeClientFrame(client *socketTestConn, opcode byte, payload []byte) error {
	mask := []byte{1, 2, 3, 4}
	header := []byte{0x80 | opcode}
	switch {
	case len(payload) < 126:
		header = append(header, 0x80|byte(len(payload)))
	case len(payload) <= 65535:
		extended := make([]byte, 2)
		binary.BigEndian.PutUint16(extended, uint16(len(payload)))
		header = append(header, 0x80|126)
		header = append(header, extended...)
	default:
		extended := make([]byte, 8)
		binary.BigEndian.PutUint64(extended, uint64(len(payload)))
		header = append(header, 0x80|127)
		header = append(header, extended...)
	}
	masked := make([]byte, len(payload))
	for i, b := range payload {
		masked[i] = b ^ mask[i%len(mask)]
	}
	frame := append(header, mask...)
	frame = append(frame, masked...)
	_, err := client.Write(frame)
	return err
}

func socketHubClientCount(hub *socketHub) int {
	hub.mu.RLock()
	defer hub.mu.RUnlock()
	return len(hub.clients)
}

func socketHubRoomClientCount(hub *socketHub, roomID string) int {
	hub.mu.RLock()
	defer hub.mu.RUnlock()
	return len(hub.rooms[roomID])
}

type socketTestHandState struct {
	RoomID           string   `json:"roomId"`
	HandID           string   `json:"handId"`
	Status           string   `json:"status"`
	CurrentSeat      int      `json:"currentSeat"`
	AvailableActions []string `json:"availableActions"`
}

type saveFailStore struct {
	Store
}

func (s *saveFailStore) Save(*game.Game) error {
	return errors.New("forced save failure")
}

func subscribeSocket(t *testing.T, client *socketTestConn, roomID, requestID string) {
	t.Helper()
	readUpgradeResponse(t, client)
	readSocketMessage(t, client, "connection.ready")
	if err := writeClientTextFrame(client, []byte(`{"type":"room.subscribe","requestId":"`+requestID+`","roomId":"`+roomID+`","payload":{}}`)); err != nil {
		t.Fatal(err)
	}
	readSocketMessage(t, client, "ack")
	readSocketMessage(t, client, "room.snapshot")
}

func startHandViaSocket(t *testing.T, ownerClient, playerClient *socketTestConn, roomID string) {
	t.Helper()
	if err := writeClientTextFrame(ownerClient, []byte(`{"type":"room.start_hand","requestId":"req_start_helper","roomId":"`+roomID+`","payload":{}}`)); err != nil {
		t.Fatal(err)
	}
	readSocketMessage(t, ownerClient, "ack")
	readSocketMessage(t, ownerClient, "hand.started")
	readSocketMessage(t, playerClient, "hand.started")
}

func currentHandViaHTTP(t *testing.T, server *Server, roomID, token string) socketTestHandState {
	t.Helper()
	state, ok := tryCurrentHandViaHTTP(t, server, roomID, token)
	if !ok {
		t.Fatalf("current hand not found for room %s", roomID)
	}
	return state
}

func tryCurrentHandViaHTTP(t *testing.T, server *Server, roomID, token string) (socketTestHandState, bool) {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, "/api/rooms/"+roomID+"/current-hand", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	res := httptest.NewRecorder()
	server.Routes().ServeHTTP(res, req)
	if res.Code == http.StatusNotFound {
		return socketTestHandState{}, false
	}
	if res.Code != http.StatusOK {
		t.Fatalf("current hand status=%d body=%s", res.Code, res.Body.String())
	}
	var state socketTestHandState
	if err := json.Unmarshal(res.Body.Bytes(), &state); err != nil {
		t.Fatal(err)
	}
	return state, true
}

func hasAction(actions []string, want string) bool {
	for _, action := range actions {
		if action == want {
			return true
		}
	}
	return false
}

func preferredAction(actions []string) string {
	switch {
	case hasAction(actions, "check"):
		return "check"
	case hasAction(actions, "call"):
		return "call"
	case hasAction(actions, "fold"):
		return "fold"
	default:
		return "all_in"
	}
}

func readSocketMessageAny(t *testing.T, client *socketTestConn) socketEnvelope {
	t.Helper()
	_, payload := readServerTextFrame(t, client)
	var msg socketEnvelope
	if err := json.Unmarshal(payload, &msg); err != nil {
		t.Fatal(err)
	}
	return msg
}

func drainWalletUpdates(t *testing.T, client *socketTestConn) int {
	t.Helper()
	count := 0
	for {
		msg, ok := tryReadSocketMessage(client, 30*time.Millisecond)
		if !ok {
			return count
		}
		if msg.Type == "wallet.updated" {
			count++
		}
	}
}

func tryReadSocketMessage(client *socketTestConn, timeout time.Duration) (socketEnvelope, bool) {
	_ = client.SetReadDeadline(time.Now().Add(timeout))
	defer client.SetReadDeadline(time.Time{})
	header := make([]byte, 2)
	if _, err := client.reader.Read(header); err != nil {
		return socketEnvelope{}, false
	}
	opcode := header[0] & 0x0f
	if opcode != 1 {
		return socketEnvelope{}, false
	}
	length := int(header[1] & 0x7f)
	if length == 126 {
		extended := make([]byte, 2)
		if _, err := client.reader.Read(extended); err != nil {
			return socketEnvelope{}, false
		}
		length = int(extended[0])<<8 | int(extended[1])
	}
	payload := make([]byte, length)
	if _, err := client.reader.Read(payload); err != nil {
		return socketEnvelope{}, false
	}
	var msg socketEnvelope
	if err := json.Unmarshal(payload, &msg); err != nil {
		return socketEnvelope{}, false
	}
	return msg, true
}

func eventually(t *testing.T, ok func() bool) {
	t.Helper()
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		if ok() {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("condition was not met before deadline")
}
