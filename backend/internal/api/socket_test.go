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

func TestSocketV11SnapshotIncludesTimerPresenceLogChatAndLeaderboard(t *testing.T) {
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

	if err := writeClientTextFrame(client, []byte(`{"type":"room.subscribe","requestId":"req_v11_snapshot","roomId":"`+roomID+`","payload":{}}`)); err != nil {
		t.Fatal(err)
	}
	readSocketMessage(t, client, "ack")
	snapshot := readSocketMessage(t, client, "room.snapshot")
	var payload struct {
		Hand *struct {
			CurrentSeat          int    `json:"currentSeat"`
			ActionStartedAt      string `json:"actionStartedAt"`
			ActionDeadlineAt     string `json:"actionDeadlineAt"`
			ActionTimeoutSeconds int    `json:"actionTimeoutSeconds"`
			ServerTime           string `json:"serverTime"`
		} `json:"hand"`
		Presence []struct {
			UserID string `json:"userId"`
			Status string `json:"status"`
		} `json:"presence"`
		RecentActionLog    []any `json:"recentActionLog"`
		RecentChatMessages []any `json:"recentChatMessages"`
		Leaderboard        []any `json:"leaderboard"`
	}
	if err := json.Unmarshal(snapshot.Payload, &payload); err != nil {
		t.Fatal(err)
	}
	if payload.Hand == nil || payload.Hand.CurrentSeat == 0 || payload.Hand.ActionStartedAt == "" || payload.Hand.ActionDeadlineAt == "" || payload.Hand.ActionTimeoutSeconds <= 0 || payload.Hand.ServerTime == "" {
		t.Fatalf("timer fields missing from hand snapshot: %#v", payload.Hand)
	}
	if len(payload.Presence) != 2 {
		t.Fatalf("presence entries = %d, want 2", len(payload.Presence))
	}
	seenOnline := false
	for _, item := range payload.Presence {
		if item.Status == "online" {
			seenOnline = true
		}
	}
	if !seenOnline {
		t.Fatalf("expected at least one online presence entry: %#v", payload.Presence)
	}
	if payload.RecentActionLog == nil {
		t.Fatal("recentActionLog should be present even when empty")
	}
	if payload.RecentChatMessages == nil {
		t.Fatal("recentChatMessages should be present even when empty")
	}
	if payload.Leaderboard == nil {
		t.Fatal("leaderboard should be present even when empty")
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

func TestSocketBroadcastsPresenceWhenMemberDisconnectsAndReconnects(t *testing.T) {
	server := testServer(t)
	roomID, ownerToken, playerToken := setupRoomWithSeats(t, server)
	ts := httptest.NewServer(server.Routes())
	defer ts.Close()
	ownerClient := dialSocket(t, ts.URL, "/api/socket?token="+ownerToken)
	defer ownerClient.Close()
	playerClient := dialSocket(t, ts.URL, "/api/socket?token="+playerToken)

	subscribeSocket(t, ownerClient, roomID, "req_owner_presence_sub")
	subscribeSocket(t, playerClient, roomID, "req_player_presence_sub")
	if err := writeClientCloseFrame(playerClient); err != nil {
		t.Fatal(err)
	}
	offlinePayload := readPresenceUpdate(t, ownerClient, func(item socketPresencePayload) bool {
		return item.Status == "offline"
	})
	if offlinePayload.Status != "offline" || offlinePayload.UserID == "" {
		t.Fatalf("offline presence payload = %#v", offlinePayload)
	}

	reconnected := dialSocket(t, ts.URL, "/api/socket?token="+playerToken)
	defer reconnected.Close()
	subscribeSocket(t, reconnected, roomID, "req_player_reconnect_sub")
	onlinePayload := readPresenceUpdate(t, ownerClient, func(item socketPresencePayload) bool {
		return item.Status == "online" && item.UserID == offlinePayload.UserID
	})
	if onlinePayload.Status != "online" || onlinePayload.UserID != offlinePayload.UserID {
		t.Fatalf("online presence payload = %#v want same user online", onlinePayload)
	}
}

func TestSocketBroadcastsPresenceWhenMemberFirstSubscribes(t *testing.T) {
	server := testServer(t)
	roomID, ownerToken, playerToken := setupRoomWithSeats(t, server)
	ts := httptest.NewServer(server.Routes())
	defer ts.Close()
	ownerClient := dialSocket(t, ts.URL, "/api/socket?token="+ownerToken)
	defer ownerClient.Close()
	playerClient := dialSocket(t, ts.URL, "/api/socket?token="+playerToken)
	defer playerClient.Close()

	subscribeSocket(t, ownerClient, roomID, "req_owner_first_presence_sub")
	readUpgradeResponse(t, playerClient)
	readSocketMessage(t, playerClient, "connection.ready")
	if err := writeClientTextFrame(playerClient, []byte(`{"type":"room.subscribe","requestId":"req_player_first_presence_sub","roomId":"`+roomID+`","payload":{}}`)); err != nil {
		t.Fatal(err)
	}
	readSocketMessage(t, playerClient, "ack")
	readSocketMessage(t, playerClient, "room.snapshot")

	payload := readPresenceUpdate(t, ownerClient, func(item socketPresencePayload) bool {
		return item.Status == "online" && item.SeatNo == 2
	})
	if payload.Status != "online" || payload.UserID == "" || payload.SeatNo != 2 {
		t.Fatalf("first subscribe presence payload = %#v, want player online at seat 2", payload)
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
	logMessage := readSocketMessage(t, actorClient, "hand.log.appended")
	var logPayload struct {
		Entry struct {
			Kind   string `json:"kind"`
			Action string `json:"action"`
			SeatNo int    `json:"seatNo"`
			Source string `json:"source"`
		} `json:"entry"`
	}
	if err := json.Unmarshal(logMessage.Payload, &logPayload); err != nil {
		t.Fatal(err)
	}
	if logPayload.Entry.Kind != "player_action" || logPayload.Entry.Action == "" || logPayload.Entry.SeatNo == 0 || logPayload.Entry.Source != "player" {
		t.Fatalf("log entry incomplete: %#v", logPayload.Entry)
	}
}

func TestSocketActionTimeoutAppliesAutomaticAction(t *testing.T) {
	server := testServer(t)
	server.actionTimeout = 20 * time.Millisecond
	roomID, ownerToken, playerToken := setupRoomWithSeats(t, server)
	ts := httptest.NewServer(server.Routes())
	defer ts.Close()
	ownerClient := dialSocket(t, ts.URL, "/api/socket?token="+ownerToken)
	defer ownerClient.Close()
	playerClient := dialSocket(t, ts.URL, "/api/socket?token="+playerToken)
	defer playerClient.Close()
	subscribeSocket(t, ownerClient, roomID, "req_owner_timeout_sub")
	subscribeSocket(t, playerClient, roomID, "req_player_timeout_sub")

	if err := writeClientTextFrame(ownerClient, []byte(`{"type":"room.start_hand","requestId":"req_timeout_start","roomId":"`+roomID+`","payload":{}}`)); err != nil {
		t.Fatal(err)
	}
	readSocketMessage(t, ownerClient, "ack")
	readSocketMessage(t, ownerClient, "hand.started")
	readSocketMessage(t, playerClient, "hand.started")
	readSocketMessage(t, ownerClient, "hand.log.appended")
	readSocketMessage(t, playerClient, "hand.log.appended")

	timeoutMsg := readSocketMessageSkipping(t, ownerClient, "hand.timeout_applied", map[string]bool{"hand.log.appended": true})
	var timeoutPayload struct {
		SeatNo int    `json:"seatNo"`
		Action string `json:"action"`
		Source string `json:"source"`
	}
	if err := json.Unmarshal(timeoutMsg.Payload, &timeoutPayload); err != nil {
		t.Fatal(err)
	}
	if timeoutPayload.SeatNo == 0 || timeoutPayload.Action == "" || timeoutPayload.Source != "timeout" {
		t.Fatalf("timeout payload incomplete: %#v", timeoutPayload)
	}
	updateMessages := readSocketMessagesUntilUpdate(t, ownerClient)
	last := updateMessages[len(updateMessages)-1]
	if last.Type == "hand.updated" {
		var payload struct {
			Hand struct {
				CurrentSeat int `json:"currentSeat"`
			} `json:"hand"`
		}
		if err := json.Unmarshal(last.Payload, &payload); err != nil {
			t.Fatal(err)
		}
		if payload.Hand.CurrentSeat == timeoutPayload.SeatNo {
			t.Fatalf("timeout did not advance current seat: timeout=%#v hand=%#v", timeoutPayload, payload.Hand)
		}
	} else if last.Type != "hand.settled" {
		t.Fatalf("timeout should update or settle hand, got %s", last.Type)
	}
}

func TestSocketActionTimeoutChecksWhenLegal(t *testing.T) {
	server := testServer(t)
	server.actionTimeout = time.Hour
	roomID, ownerToken, playerToken := setupRoomWithSeats(t, server)
	ts := httptest.NewServer(server.Routes())
	defer ts.Close()
	ownerClient := dialSocket(t, ts.URL, "/api/socket?token="+ownerToken)
	defer ownerClient.Close()
	playerClient := dialSocket(t, ts.URL, "/api/socket?token="+playerToken)
	defer playerClient.Close()
	subscribeSocket(t, ownerClient, roomID, "req_owner_timeout_check_sub")
	subscribeSocket(t, playerClient, roomID, "req_player_timeout_check_sub")

	startReq := httptest.NewRequest(http.MethodPost, "/api/rooms/"+roomID+"/start", nil)
	startReq.Header.Set("Authorization", "Bearer "+ownerToken)
	startRes := httptest.NewRecorder()
	server.Routes().ServeHTTP(startRes, startReq)
	if startRes.Code != http.StatusOK {
		t.Fatalf("start status=%d body=%s", startRes.Code, startRes.Body.String())
	}

	var current socketTestHandState
	for i := 0; i < 8; i++ {
		current = currentHandViaHTTP(t, server, roomID, ownerToken)
		if hasAction(current.AvailableActions, "check") {
			break
		}
		action := preferredAction(current.AvailableActions)
		token := ownerToken
		if current.CurrentSeat == 2 {
			token = playerToken
		}
		actReq := httptest.NewRequest(http.MethodPost, "/api/rooms/"+roomID+"/actions", bytes.NewReader([]byte(`{"action":"`+action+`","amount":0}`)))
		actReq.Header.Set("Authorization", "Bearer "+token)
		actRes := httptest.NewRecorder()
		server.Routes().ServeHTTP(actRes, actReq)
		if actRes.Code != http.StatusOK {
			t.Fatalf("action status=%d body=%s", actRes.Code, actRes.Body.String())
		}
	}
	if !hasAction(current.AvailableActions, "check") {
		t.Fatalf("did not reach check-capable state: %#v", current)
	}
	server.actionTimeout = 20 * time.Millisecond
	server.scheduleActionTimeout(roomID, current.HandID, current.CurrentSeat)

	timeoutMsg := readSocketMessageSkipping(t, ownerClient, "hand.timeout_applied", map[string]bool{"hand.log.appended": true, "hand.updated": true})
	var timeoutPayload struct {
		Action string `json:"action"`
		Source string `json:"source"`
	}
	if err := json.Unmarshal(timeoutMsg.Payload, &timeoutPayload); err != nil {
		t.Fatal(err)
	}
	if timeoutPayload.Action != "check" || timeoutPayload.Source != "timeout" {
		t.Fatalf("timeout payload = %#v, want timeout check", timeoutPayload)
	}
}

func TestHTTPSeatChangesBroadcastRoomUpdateAndActionLog(t *testing.T) {
	server := testServer(t)
	ownerToken := registerUser(t, server, "seatlog_owner", "座位房主")
	playerToken := registerUser(t, server, "seatlog_player", "座位玩家")

	createReq := httptest.NewRequest(http.MethodPost, "/api/rooms", bytes.NewReader([]byte(`{"ruleSetId":"long-holdem","seatCount":6,"minPlayersToStart":2}`)))
	createReq.Header.Set("Authorization", "Bearer "+ownerToken)
	createRes := httptest.NewRecorder()
	server.Routes().ServeHTTP(createRes, createReq)
	if createRes.Code != http.StatusCreated {
		t.Fatalf("create room status=%d body=%s", createRes.Code, createRes.Body.String())
	}
	var room struct {
		ID         string `json:"id"`
		InviteCode string `json:"inviteCode"`
	}
	if err := json.Unmarshal(createRes.Body.Bytes(), &room); err != nil {
		t.Fatal(err)
	}

	ts := httptest.NewServer(server.Routes())
	defer ts.Close()
	ownerClient := dialSocket(t, ts.URL, "/api/socket?token="+ownerToken)
	defer ownerClient.Close()
	subscribeSocket(t, ownerClient, room.ID, "req_owner_seatlog_sub")

	joinReq := httptest.NewRequest(http.MethodPost, "/api/rooms/join", bytes.NewReader([]byte(`{"inviteCode":"`+room.InviteCode+`"}`)))
	joinReq.Header.Set("Authorization", "Bearer "+playerToken)
	joinRes := httptest.NewRecorder()
	server.Routes().ServeHTTP(joinRes, joinReq)
	if joinRes.Code != http.StatusOK {
		t.Fatalf("join status=%d body=%s", joinRes.Code, joinRes.Body.String())
	}
	readSocketMessage(t, ownerClient, "room.updated")

	takeReq := httptest.NewRequest(http.MethodPost, "/api/rooms/"+room.ID+"/seats/2", bytes.NewReader([]byte(`{"buyInChips":1000}`)))
	takeReq.Header.Set("Authorization", "Bearer "+playerToken)
	takeRes := httptest.NewRecorder()
	server.Routes().ServeHTTP(takeRes, takeReq)
	if takeRes.Code != http.StatusOK {
		t.Fatalf("take seat status=%d body=%s", takeRes.Code, takeRes.Body.String())
	}
	readSocketMessage(t, ownerClient, "room.updated")
	takeLog := readSocketMessage(t, ownerClient, "hand.log.appended")
	var takePayload struct {
		Entry struct {
			Kind   string `json:"kind"`
			SeatNo int    `json:"seatNo"`
		} `json:"entry"`
	}
	if err := json.Unmarshal(takeLog.Payload, &takePayload); err != nil {
		t.Fatal(err)
	}
	if takePayload.Entry.Kind != "seat_taken" || takePayload.Entry.SeatNo != 2 {
		t.Fatalf("take seat log = %#v", takePayload.Entry)
	}

	leaveReq := httptest.NewRequest(http.MethodDelete, "/api/rooms/"+room.ID+"/seats/2", nil)
	leaveReq.Header.Set("Authorization", "Bearer "+playerToken)
	leaveRes := httptest.NewRecorder()
	server.Routes().ServeHTTP(leaveRes, leaveReq)
	if leaveRes.Code != http.StatusOK {
		t.Fatalf("leave seat status=%d body=%s", leaveRes.Code, leaveRes.Body.String())
	}
	readSocketMessage(t, ownerClient, "room.updated")
	leaveLog := readSocketMessage(t, ownerClient, "hand.log.appended")
	var leavePayload struct {
		Entry struct {
			Kind   string `json:"kind"`
			SeatNo int    `json:"seatNo"`
		} `json:"entry"`
	}
	if err := json.Unmarshal(leaveLog.Payload, &leavePayload); err != nil {
		t.Fatal(err)
	}
	if leavePayload.Entry.Kind != "seat_left" || leavePayload.Entry.SeatNo != 2 {
		t.Fatalf("leave seat log = %#v", leavePayload.Entry)
	}
}

func TestSocketChatSendBroadcastsTextAndRejectsInvalidMessages(t *testing.T) {
	server := testServer(t)
	roomID, ownerToken, playerToken := setupRoomWithSeats(t, server)
	ts := httptest.NewServer(server.Routes())
	defer ts.Close()
	ownerClient := dialSocket(t, ts.URL, "/api/socket?token="+ownerToken)
	defer ownerClient.Close()
	playerClient := dialSocket(t, ts.URL, "/api/socket?token="+playerToken)
	defer playerClient.Close()
	subscribeSocket(t, ownerClient, roomID, "req_owner_chat_sub")
	subscribeSocket(t, playerClient, roomID, "req_player_chat_sub")

	if err := writeClientTextFrame(ownerClient, []byte(`{"type":"chat.send","requestId":"req_chat_text","roomId":"`+roomID+`","payload":{"kind":"text","text":"这手精彩"}}`)); err != nil {
		t.Fatal(err)
	}
	ack := readSocketMessage(t, ownerClient, "ack")
	if ack.RequestID != "req_chat_text" {
		t.Fatalf("chat ack requestId = %s, want req_chat_text", ack.RequestID)
	}
	for _, client := range []*socketTestConn{ownerClient, playerClient} {
		msg := readSocketMessage(t, client, "chat.message")
		var payload struct {
			Message struct {
				ID       string `json:"id"`
				Kind     string `json:"kind"`
				Text     string `json:"text"`
				Nickname string `json:"nickname"`
			} `json:"message"`
		}
		if err := json.Unmarshal(msg.Payload, &payload); err != nil {
			t.Fatal(err)
		}
		if payload.Message.ID == "" || payload.Message.Kind != "text" || payload.Message.Text != "这手精彩" || payload.Message.Nickname == "" {
			t.Fatalf("chat message payload incomplete: %#v", payload.Message)
		}
	}

	if err := writeClientTextFrame(ownerClient, []byte(`{"type":"chat.send","requestId":"req_bad_emoji","roomId":"`+roomID+`","payload":{"kind":"emoji","emojiCode":"not_allowed"}}`)); err != nil {
		t.Fatal(err)
	}
	msg := readSocketMessage(t, ownerClient, "error")
	var payload ErrorResponse
	if err := json.Unmarshal(msg.Payload, &payload); err != nil {
		t.Fatal(err)
	}
	if payload.Code != "chat_emoji_unknown" {
		t.Fatalf("error code = %s, want chat_emoji_unknown", payload.Code)
	}
}

func TestSocketChatSendRejectsNonMemberTooLongAndRateLimitedMessages(t *testing.T) {
	server := testServer(t)
	roomID, ownerToken, _ := setupRoomWithSeats(t, server)
	outsiderToken := registerUser(t, server, "chat_outsider", "聊天旁观")
	ts := httptest.NewServer(server.Routes())
	defer ts.Close()

	outsiderClient := dialSocket(t, ts.URL, "/api/socket?token="+outsiderToken)
	defer outsiderClient.Close()
	readUpgradeResponse(t, outsiderClient)
	readSocketMessage(t, outsiderClient, "connection.ready")
	if err := writeClientTextFrame(outsiderClient, []byte(`{"type":"chat.send","requestId":"req_chat_outsider","roomId":"`+roomID+`","payload":{"kind":"text","text":"hello"}}`)); err != nil {
		t.Fatal(err)
	}
	outsiderError := readSocketMessage(t, outsiderClient, "error")
	var outsiderPayload ErrorResponse
	if err := json.Unmarshal(outsiderError.Payload, &outsiderPayload); err != nil {
		t.Fatal(err)
	}
	if outsiderPayload.Code != "forbidden" {
		t.Fatalf("outsider chat error code = %s, want forbidden", outsiderPayload.Code)
	}

	ownerClient := dialSocket(t, ts.URL, "/api/socket?token="+ownerToken)
	defer ownerClient.Close()
	subscribeSocket(t, ownerClient, roomID, "req_owner_chat_limits_sub")
	longText := strings.Repeat("长", 201)
	if err := writeClientTextFrame(ownerClient, []byte(`{"type":"chat.send","requestId":"req_chat_long","roomId":"`+roomID+`","payload":{"kind":"text","text":"`+longText+`"}}`)); err != nil {
		t.Fatal(err)
	}
	longError := readSocketMessage(t, ownerClient, "error")
	var longPayload ErrorResponse
	if err := json.Unmarshal(longError.Payload, &longPayload); err != nil {
		t.Fatal(err)
	}
	if longPayload.Code != "chat_message_too_long" {
		t.Fatalf("long chat error code = %s, want chat_message_too_long", longPayload.Code)
	}

	if err := writeClientTextFrame(ownerClient, []byte(`{"type":"chat.send","requestId":"req_chat_first","roomId":"`+roomID+`","payload":{"kind":"text","text":"第一条"}}`)); err != nil {
		t.Fatal(err)
	}
	readSocketMessage(t, ownerClient, "ack")
	readSocketMessage(t, ownerClient, "chat.message")
	if err := writeClientTextFrame(ownerClient, []byte(`{"type":"chat.send","requestId":"req_chat_fast","roomId":"`+roomID+`","payload":{"kind":"text","text":"太快了"}}`)); err != nil {
		t.Fatal(err)
	}
	rateError := readSocketMessage(t, ownerClient, "error")
	var ratePayload ErrorResponse
	if err := json.Unmarshal(rateError.Payload, &ratePayload); err != nil {
		t.Fatal(err)
	}
	if ratePayload.Code != "chat_rate_limited" {
		t.Fatalf("rate limit error code = %s, want chat_rate_limited", ratePayload.Code)
	}
}

func TestSocketChatSendBroadcastsAllowedEmoji(t *testing.T) {
	server := testServer(t)
	roomID, ownerToken, playerToken := setupRoomWithSeats(t, server)
	ts := httptest.NewServer(server.Routes())
	defer ts.Close()
	ownerClient := dialSocket(t, ts.URL, "/api/socket?token="+ownerToken)
	defer ownerClient.Close()
	playerClient := dialSocket(t, ts.URL, "/api/socket?token="+playerToken)
	defer playerClient.Close()
	subscribeSocket(t, ownerClient, roomID, "req_owner_chat_emoji_sub")
	subscribeSocket(t, playerClient, roomID, "req_player_chat_emoji_sub")

	if err := writeClientTextFrame(ownerClient, []byte(`{"type":"chat.send","requestId":"req_chat_emoji","roomId":"`+roomID+`","payload":{"kind":"emoji","emojiCode":"nice_hand"}}`)); err != nil {
		t.Fatal(err)
	}
	if ack := readSocketMessage(t, ownerClient, "ack"); ack.RequestID != "req_chat_emoji" {
		t.Fatalf("emoji ack requestId = %s, want req_chat_emoji", ack.RequestID)
	}
	for _, client := range []*socketTestConn{ownerClient, playerClient} {
		msg := readSocketMessage(t, client, "chat.message")
		var payload struct {
			Message struct {
				Kind      string `json:"kind"`
				EmojiCode string `json:"emojiCode"`
			} `json:"message"`
		}
		if err := json.Unmarshal(msg.Payload, &payload); err != nil {
			t.Fatal(err)
		}
		if payload.Message.Kind != "emoji" || payload.Message.EmojiCode != "nice_hand" {
			t.Fatalf("emoji chat payload = %#v, want nice_hand emoji", payload.Message)
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
		readSocketMessageSkipping(t, actor, "ack", map[string]bool{"hand.log.appended": true})
		for _, msg := range readSocketMessagesUntilUpdate(t, actor) {
			if msg.Type == "hand.settled" {
				settled = msg
			}
			if msg.Type == "wallet.updated" && actor == ownerClient {
				ownerWalletUpdates++
			}
			if msg.Type == "wallet.updated" && actor == playerClient {
				playerWalletUpdates++
			}
		}
		for _, msg := range readSocketMessagesUntilUpdate(t, observer) {
			if msg.Type == "hand.settled" {
				settled = msg
			}
			if msg.Type == "wallet.updated" && observer == ownerClient {
				ownerWalletUpdates++
			}
			if msg.Type == "wallet.updated" && observer == playerClient {
				playerWalletUpdates++
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

func TestSocketSettlementBroadcastsRoomUpdateAndAutoStartsNextHand(t *testing.T) {
	server := testServer(t)
	server.actionTimeout = 5 * time.Second
	server.autoNextDelay = 20 * time.Millisecond
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

	var sawSettled bool
	for i := 0; i < 12; i++ {
		current, ok := tryCurrentHandViaHTTP(t, server, roomID, ownerToken)
		if !ok {
			break
		}
		action := preferredAction(current.AvailableActions)
		actor := ownerClient
		if current.CurrentSeat == 2 {
			actor = playerClient
		}
		if err := writeClientTextFrame(actor, []byte(`{"type":"room.action","requestId":"req_auto_next_`+itoa(i)+`","roomId":"`+roomID+`","payload":{"action":"`+action+`","amount":0}}`)); err != nil {
			t.Fatal(err)
		}
		readSocketMessageSkipping(t, actor, "ack", map[string]bool{
			"hand.log.appended":        true,
			"hand.updated":             true,
			"hand.settled":             true,
			"wallet.updated":           true,
			"room.leaderboard.updated": true,
			"room.updated":             true,
		})
		for _, msg := range readSocketMessagesUntilUpdate(t, actor) {
			if msg.Type == "hand.settled" {
				sawSettled = true
			}
		}
		if sawSettled {
			break
		}
	}
	if !sawSettled {
		t.Fatal("expected hand.settled before auto-start verification")
	}

	roomUpdate := readSocketMessageSkipping(t, ownerClient, "room.updated", map[string]bool{
		"hand.log.appended":        true,
		"wallet.updated":           true,
		"room.leaderboard.updated": true,
		"hand.settled":             true,
		"player.presence.updated":  true,
	})
	var updatedPayload struct {
		Room struct {
			ID     string `json:"id"`
			Status string `json:"status"`
		} `json:"room"`
	}
	if err := json.Unmarshal(roomUpdate.Payload, &updatedPayload); err != nil {
		t.Fatal(err)
	}
	if updatedPayload.Room.ID != roomID || updatedPayload.Room.Status != "waiting" {
		t.Fatalf("room.updated payload=%#v, want waiting room %s", updatedPayload.Room, roomID)
	}

	nextStarted := readSocketMessageSkipping(t, ownerClient, "hand.started", map[string]bool{
		"hand.log.appended":        true,
		"wallet.updated":           true,
		"room.leaderboard.updated": true,
		"room.updated":             true,
	})
	var nextPayload struct {
		Hand struct {
			HandID string `json:"handId"`
		} `json:"hand"`
	}
	if err := json.Unmarshal(nextStarted.Payload, &nextPayload); err != nil {
		t.Fatal(err)
	}
	if nextPayload.Hand.HandID == "" {
		t.Fatalf("next hand payload incomplete: %#v", nextPayload.Hand)
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
	for i := 0; i < 10; i++ {
		msg := readSocketMessageAny(t, client)
		if msg.Type == wantType {
			return msg
		}
		if msg.Type == "player.presence.updated" && wantType != "player.presence.updated" {
			continue
		}
		t.Fatalf("message type = %s, want %s; payload=%s", msg.Type, wantType, string(msg.Payload))
	}
	t.Fatalf("did not receive %s after skipped messages", wantType)
	return socketEnvelope{}
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

type socketPresencePayload struct {
	Status string `json:"status"`
	UserID string `json:"userId"`
	SeatNo int    `json:"seatNo"`
}

func readPresenceUpdate(t *testing.T, client *socketTestConn, matches func(socketPresencePayload) bool) socketPresencePayload {
	t.Helper()
	for i := 0; i < 10; i++ {
		msg := readSocketMessageAny(t, client)
		if msg.Type != "player.presence.updated" {
			t.Fatalf("message type = %s, want player.presence.updated; payload=%s", msg.Type, string(msg.Payload))
		}
		var payload socketPresencePayload
		if err := json.Unmarshal(msg.Payload, &payload); err != nil {
			t.Fatal(err)
		}
		if matches(payload) {
			return payload
		}
	}
	t.Fatal("matching presence update not received")
	return socketPresencePayload{}
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
	readSocketMessage(t, ownerClient, "hand.log.appended")
	readSocketMessage(t, playerClient, "hand.log.appended")
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

func readSocketMessageSkipping(t *testing.T, client *socketTestConn, wantType string, skip map[string]bool) socketEnvelope {
	t.Helper()
	for i := 0; i < 10; i++ {
		msg := readSocketMessageAny(t, client)
		if msg.Type == wantType {
			return msg
		}
		if skip[msg.Type] || msg.Type == "player.presence.updated" {
			continue
		}
		t.Fatalf("message type = %s, want %s; envelope=%#v", msg.Type, wantType, msg)
	}
	t.Fatalf("did not receive %s after skipped messages", wantType)
	return socketEnvelope{}
}

func readSocketMessagesUntilUpdate(t *testing.T, client *socketTestConn) []socketEnvelope {
	t.Helper()
	messages := []socketEnvelope{}
	for i := 0; i < 10; i++ {
		msg := readSocketMessageAny(t, client)
		messages = append(messages, msg)
		if msg.Type == "hand.updated" || msg.Type == "hand.settled" {
			return messages
		}
	}
	t.Fatalf("did not receive hand update or settlement; messages=%#v", messages)
	return messages
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
