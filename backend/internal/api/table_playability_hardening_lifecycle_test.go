package api

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/dengbin9009/DePu/backend/internal/game"
	"github.com/dengbin9009/DePu/backend/internal/storage"
)

func TestTablePlayabilityHardeningLifecycleMatrix(t *testing.T) {
	t.Run("http start serializes with room lifecycle commands", testLifecycleHTTPStartSerializesWithRoomCommands)
	t.Run("http action serializes with room lifecycle commands", testLifecycleHTTPActionSerializesWithRoomCommands)
	t.Run("settlement persists before ordered broadcasts", testLifecycleSettlementEventOrder)
	t.Run("settlement broadcast does not depend on post-commit reload", testLifecycleSettlementBroadcastWithoutReload)
	t.Run("auto starts next hand from authoritative waiting room", testLifecycleAutoNextHand)
	t.Run("auto next uses authoritative owner after delay", testLifecycleAutoNextUsesAuthoritativeOwner)
	t.Run("auto next stops when seated players are insufficient", testLifecycleAutoNextInsufficientPlayers)
	t.Run("auto next stops when room closes during delay", testLifecycleAutoNextClosedRoom)
	t.Run("auto next preserves an authoritative hand started during delay", testLifecycleAutoNextExistingHand)
	t.Run("duplicate delayed tasks create only one hand", testLifecycleDuplicateAutoNextTasks)
	t.Run("owner transfers by circular seated order", testLifecycleOwnerTransferOrder)
	t.Run("owner leave closes room without successor", testLifecycleOwnerLeaveClosesRoom)
	t.Run("owner transfer broadcasts and changes command authority", testLifecycleOwnerTransferBroadcastAndCommandAuthority)
	t.Run("player leave broadcasts authoritative room", testLifecyclePlayerLeaveBroadcast)
	t.Run("room close broadcasts authoritative room", testLifecycleRoomCloseBroadcast)
}

func testLifecycleHTTPActionSerializesWithRoomCommands(t *testing.T) {
	base := testServer(t)
	roomID, ownerToken, playerToken := setupRoomWithSeats(t, base)
	startRequest := httptest.NewRequest(http.MethodPost, "/api/rooms/"+roomID+"/start", nil)
	startRequest.Header.Set("Authorization", "Bearer "+ownerToken)
	startResponse := httptest.NewRecorder()
	base.Routes().ServeHTTP(startResponse, startRequest)
	if startResponse.Code != http.StatusOK {
		t.Fatalf("start room status=%d body=%s", startResponse.Code, startResponse.Body.String())
	}
	current := currentHandViaHTTP(t, base, roomID, ownerToken)
	actorToken := ownerToken
	if current.CurrentSeat == 2 {
		actorToken = playerToken
	}

	store := newBlockingFirstSaveStore(base.store)
	server := NewServerWithStore(store)
	server.sessions = base.sessions
	results := make(chan *httptest.ResponseRecorder, 2)
	action := preferredAction(current.AvailableActions)
	submit := func() {
		request := httptest.NewRequest(http.MethodPost, "/api/rooms/"+roomID+"/actions", bytes.NewBufferString(`{"action":"`+action+`","amount":0}`))
		request.Header.Set("Authorization", "Bearer "+actorToken)
		response := httptest.NewRecorder()
		server.Routes().ServeHTTP(response, request)
		results <- response
	}

	go submit()
	select {
	case <-store.firstSaveStarted:
	case <-time.After(time.Second):
		t.Fatal("first HTTP action did not reach game persistence")
	}

	go submit()
	select {
	case response := <-results:
		close(store.releaseFirstSave)
		t.Fatalf("second HTTP action completed before first lifecycle command: status=%d body=%s", response.Code, response.Body.String())
	case <-time.After(100 * time.Millisecond):
	}

	close(store.releaseFirstSave)
	first := <-results
	second := <-results
	if first.Code != http.StatusOK && second.Code != http.StatusOK {
		t.Fatalf("HTTP action statuses=(%d,%d), want one successful action", first.Code, second.Code)
	}
	if first.Code == http.StatusOK && second.Code == http.StatusOK {
		t.Fatalf("HTTP action statuses=(%d,%d), duplicate action unexpectedly succeeded", first.Code, second.Code)
	}
}

func testLifecycleHTTPStartSerializesWithRoomCommands(t *testing.T) {
	base := testServer(t)
	roomID, ownerToken, _ := setupRoomWithSeats(t, base)
	store := newBlockingFirstSaveStore(base.store)
	server := NewServerWithStore(store)
	server.sessions = base.sessions

	results := make(chan *httptest.ResponseRecorder, 2)
	start := func() {
		request := httptest.NewRequest(http.MethodPost, "/api/rooms/"+roomID+"/start", nil)
		request.Header.Set("Authorization", "Bearer "+ownerToken)
		response := httptest.NewRecorder()
		server.Routes().ServeHTTP(response, request)
		results <- response
	}

	go start()
	select {
	case <-store.firstSaveStarted:
	case <-time.After(time.Second):
		t.Fatal("first HTTP start did not reach game persistence")
	}

	go start()
	select {
	case response := <-results:
		close(store.releaseFirstSave)
		t.Fatalf("second HTTP start completed before first lifecycle command: status=%d body=%s", response.Code, response.Body.String())
	case <-time.After(100 * time.Millisecond):
	}

	close(store.releaseFirstSave)
	first := <-results
	second := <-results
	statusCounts := map[int]int{first.Code: 1}
	statusCounts[second.Code]++
	if statusCounts[http.StatusOK] != 1 || statusCounts[http.StatusConflict] != 1 {
		t.Fatalf("HTTP start statuses=(%d,%d), want one 200 and one 409", first.Code, second.Code)
	}
}

func testLifecycleSettlementBroadcastWithoutReload(t *testing.T) {
	base := testServer(t)
	roomID, ownerToken, playerToken := setupRoomWithSeats(t, base)
	server := NewServerWithStore(&recentHandResultsFailStore{Store: base.store})
	server.sessions = base.sessions
	server.actionTimeout = 5 * time.Second
	server.autoNextDelay = time.Second
	ts := httptest.NewServer(server.Routes())
	defer ts.Close()

	ownerClient := dialSocket(t, ts.URL, "/api/socket?token="+ownerToken)
	defer ownerClient.Close()
	playerClient := dialSocket(t, ts.URL, "/api/socket?token="+playerToken)
	defer playerClient.Close()
	subscribeSocket(t, ownerClient, roomID, "req_lifecycle_reload_owner_sub")
	subscribeSocket(t, playerClient, roomID, "req_lifecycle_reload_player_sub")
	startHandViaSocket(t, ownerClient, playerClient, roomID)

	current := currentHandViaHTTP(t, server, roomID, ownerToken)
	actor := ownerClient
	observer := playerClient
	if current.CurrentSeat == 2 {
		actor = playerClient
		observer = ownerClient
	}
	if err := writeClientTextFrame(actor, []byte(`{"type":"room.action","requestId":"req_lifecycle_reload_settle","roomId":"`+roomID+`","payload":{"action":"fold","amount":0}}`)); err != nil {
		t.Fatal(err)
	}
	readSocketMessageSkipping(t, actor, "ack", map[string]bool{"hand.log.appended": true, "player.presence.updated": true})

	room, err := base.store.RoomByID(roomID)
	if err != nil {
		t.Fatal(err)
	}
	if room.Status != "waiting" || room.CurrentGameID != "" {
		t.Fatalf("room after committed settlement = %#v, want waiting without current game", room)
	}
	hands, err := base.store.RecentHandResultsByRoom(roomID, 1)
	if err != nil {
		t.Fatal(err)
	}
	if len(hands) != 1 || hands[0].GameID != current.HandID {
		t.Fatalf("persisted hands = %#v, want settled game %s", hands, current.HandID)
	}
	readSocketMessageSkipping(t, observer, "hand.settled", map[string]bool{
		"hand.log.appended":       true,
		"player.presence.updated": true,
	})
}

func TestLifecycleEventOrderAssertionRejectsRegression(t *testing.T) {
	err := assertLifecycleEventOrder([]string{
		"room.updated",
		"hand.settled",
		"wallet.updated",
		"room.leaderboard.updated",
	})
	if err == nil {
		t.Fatal("expected out-of-order lifecycle events to be rejected")
	}
}

func testLifecycleSettlementEventOrder(t *testing.T) {
	server := testServer(t)
	server.actionTimeout = 5 * time.Second
	server.autoNextDelay = time.Second
	roomID, ownerToken, playerToken := setupRoomWithSeats(t, server)
	ts := httptest.NewServer(server.Routes())
	defer ts.Close()

	ownerClient := dialSocket(t, ts.URL, "/api/socket?token="+ownerToken)
	defer ownerClient.Close()
	playerClient := dialSocket(t, ts.URL, "/api/socket?token="+playerToken)
	defer playerClient.Close()
	subscribeSocket(t, ownerClient, roomID, "req_lifecycle_owner_sub")
	subscribeSocket(t, playerClient, roomID, "req_lifecycle_player_sub")
	startHandViaSocket(t, ownerClient, playerClient, roomID)

	current := currentHandViaHTTP(t, server, roomID, ownerToken)
	actor := ownerClient
	observer := playerClient
	if current.CurrentSeat == 2 {
		actor = playerClient
		observer = ownerClient
	}
	if err := writeClientTextFrame(actor, []byte(`{"type":"room.action","requestId":"req_lifecycle_settle","roomId":"`+roomID+`","payload":{"action":"fold","amount":0}}`)); err != nil {
		t.Fatal(err)
	}
	readSocketMessageSkipping(t, actor, "ack", map[string]bool{"player.presence.updated": true})

	eventTypes := make([]string, 0, 4)
	for len(eventTypes) < 4 {
		message := readSocketMessageAny(t, observer)
		switch message.Type {
		case "hand.settled", "wallet.updated", "room.leaderboard.updated", "room.updated":
			eventTypes = append(eventTypes, message.Type)
		}
		if len(eventTypes) == 1 {
			room, err := server.store.RoomByID(roomID)
			if err != nil {
				t.Fatal(err)
			}
			if room.Status != "waiting" || room.CurrentGameID != "" {
				t.Fatalf("room at settlement broadcast = %#v, want persisted waiting state", room)
			}
			hands, err := server.store.RecentHandResultsByRoom(roomID, 1)
			if err != nil {
				t.Fatal(err)
			}
			if len(hands) != 1 || hands[0].GameID != current.HandID {
				t.Fatalf("persisted hands = %#v, want settled game %s", hands, current.HandID)
			}
		}
	}
	if err := assertLifecycleEventOrder(eventTypes); err != nil {
		t.Fatal(err)
	}
}

func testLifecycleAutoNextHand(t *testing.T) {
	server := testServer(t)
	server.actionTimeout = 5 * time.Second
	server.autoNextDelay = 20 * time.Millisecond
	roomID, ownerToken, _ := setupRoomWithSeats(t, server)
	ts := httptest.NewServer(server.Routes())
	defer ts.Close()
	ownerClient := dialSocket(t, ts.URL, "/api/socket?token="+ownerToken)
	defer ownerClient.Close()
	subscribeSocket(t, ownerClient, roomID, "req_lifecycle_auto_sub")

	server.scheduleAutoNextHand(roomID)
	started := readSocketMessage(t, ownerClient, "hand.started")
	var payload struct {
		Hand struct {
			HandID string `json:"handId"`
		} `json:"hand"`
	}
	if err := json.Unmarshal(started.Payload, &payload); err != nil {
		t.Fatal(err)
	}
	if payload.Hand.HandID == "" {
		t.Fatal("auto next hand broadcast missing handId")
	}
	eventually(t, func() bool {
		updated, roomErr := server.store.RoomByID(roomID)
		return roomErr == nil && updated.Status == "playing" && updated.CurrentGameID == payload.Hand.HandID
	})
}

func testLifecycleAutoNextUsesAuthoritativeOwner(t *testing.T) {
	base := testServer(t)
	roomID, ownerToken, playerToken := setupRoomWithSeats(t, base)
	store := &authoritativeRoomOwnerStore{Store: base.store, roomID: roomID}
	server := NewServerWithStore(store)
	server.sessions = base.sessions
	server.autoNextDelay = 40 * time.Millisecond
	ts := httptest.NewServer(server.Routes())
	defer ts.Close()
	ownerClient := dialSocket(t, ts.URL, "/api/socket?token="+ownerToken)
	defer ownerClient.Close()
	subscribeSocket(t, ownerClient, roomID, "req_lifecycle_authoritative_owner_sub")

	server.scheduleAutoNextHand(roomID)
	store.setOwner(lifecycleUserID(t, server, playerToken))

	started := readSocketMessage(t, ownerClient, "hand.started")
	var payload struct {
		Hand struct {
			HandID string `json:"handId"`
		} `json:"hand"`
	}
	if err := json.Unmarshal(started.Payload, &payload); err != nil {
		t.Fatal(err)
	}
	if payload.Hand.HandID == "" {
		t.Fatal("auto next hand broadcast missing handId after owner change")
	}
}

func testLifecycleAutoNextInsufficientPlayers(t *testing.T) {
	server := testServer(t)
	server.autoNextDelay = 80 * time.Millisecond
	roomID, ownerToken, playerToken := setupRoomWithSeats(t, server)
	ts := httptest.NewServer(server.Routes())
	defer ts.Close()
	ownerClient := dialSocket(t, ts.URL, "/api/socket?token="+ownerToken)
	defer ownerClient.Close()
	subscribeSocket(t, ownerClient, roomID, "req_lifecycle_insufficient_sub")

	server.scheduleAutoNextHand(roomID)
	standUp := httptest.NewRequest(http.MethodDelete, "/api/rooms/"+roomID+"/seats/2", nil)
	standUp.Header.Set("Authorization", "Bearer "+playerToken)
	standUpResult := httptest.NewRecorder()
	server.Routes().ServeHTTP(standUpResult, standUp)
	if standUpResult.Code != http.StatusOK {
		t.Fatalf("stand up status=%d body=%s", standUpResult.Code, standUpResult.Body.String())
	}
	assertNoSocketEventType(t, ownerClient, "hand.started", 160*time.Millisecond)
	updated, err := server.store.RoomByID(roomID)
	if err != nil {
		t.Fatal(err)
	}
	if updated.Status != "waiting" || updated.CurrentGameID != "" {
		t.Fatalf("room after cancelled auto next = %#v, want waiting without current game", updated)
	}
}

func testLifecycleAutoNextClosedRoom(t *testing.T) {
	server := testServer(t)
	server.autoNextDelay = 100 * time.Millisecond
	roomID, ownerToken, playerToken := setupRoomWithSeats(t, server)
	ts := httptest.NewServer(server.Routes())
	defer ts.Close()
	ownerClient := dialSocket(t, ts.URL, "/api/socket?token="+ownerToken)
	defer ownerClient.Close()
	subscribeSocket(t, ownerClient, roomID, "req_lifecycle_closed_sub")

	server.scheduleAutoNextHand(roomID)
	leaveLifecycleRoom(t, server, ownerToken, roomID)
	leaveLifecycleRoom(t, server, playerToken, roomID)

	assertNoSocketEventType(t, ownerClient, "hand.started", 180*time.Millisecond)
	updated, err := server.store.RoomByID(roomID)
	if err != nil {
		t.Fatal(err)
	}
	if updated.Status != "closed" || updated.CurrentGameID != "" {
		t.Fatalf("room after cancelled auto next = %#v, want closed without current game", updated)
	}
}

func testLifecycleAutoNextExistingHand(t *testing.T) {
	base := testServer(t)
	roomID, ownerToken, _ := setupRoomWithSeats(t, base)
	existingHandIDs := make(chan string, 1)
	store := &roomReadMutationStore{Store: base.store, roomID: roomID}
	server := NewServerWithStore(store)
	server.sessions = base.sessions
	server.autoNextDelay = 20 * time.Millisecond
	ts := httptest.NewServer(server.Routes())
	defer ts.Close()
	ownerClient := dialSocket(t, ts.URL, "/api/socket?token="+ownerToken)
	defer ownerClient.Close()
	subscribeSocket(t, ownerClient, roomID, "req_lifecycle_existing_hand_sub")
	ownerUserID := lifecycleUserID(t, server, ownerToken)
	store.arm(func() error {
		result, apiErr := base.startRoomHandForUser(roomID, ownerUserID)
		if apiErr != nil {
			return errors.New(apiErr.Code + ": " + apiErr.Message)
		}
		existingHandIDs <- stringFromMap(result.State, "handId")
		return nil
	})

	server.scheduleAutoNextHand(roomID)
	var existingHandID string
	select {
	case existingHandID = <-existingHandIDs:
	case <-time.After(time.Second):
		t.Fatal("authoritative hand was not started during auto-next recheck")
	}
	assertNoSocketEventType(t, ownerClient, "hand.started", 160*time.Millisecond)
	updated, err := base.store.RoomByID(roomID)
	if err != nil {
		t.Fatal(err)
	}
	if existingHandID == "" || updated.Status != "playing" || updated.CurrentGameID != existingHandID {
		t.Fatalf("room after competing start = %#v, want authoritative hand %s", updated, existingHandID)
	}
}

func testLifecycleDuplicateAutoNextTasks(t *testing.T) {
	server := testServer(t)
	server.actionTimeout = 5 * time.Second
	server.autoNextDelay = 20 * time.Millisecond
	roomID, ownerToken, _ := setupRoomWithSeats(t, server)
	ts := httptest.NewServer(server.Routes())
	defer ts.Close()
	ownerClient := dialSocket(t, ts.URL, "/api/socket?token="+ownerToken)
	defer ownerClient.Close()
	subscribeSocket(t, ownerClient, roomID, "req_lifecycle_duplicate_sub")

	server.scheduleAutoNextHand(roomID)
	server.scheduleAutoNextHand(roomID)
	started := readSocketMessage(t, ownerClient, "hand.started")
	var payload struct {
		Hand struct {
			HandID string `json:"handId"`
		} `json:"hand"`
	}
	if err := json.Unmarshal(started.Payload, &payload); err != nil {
		t.Fatal(err)
	}
	readSocketMessage(t, ownerClient, "hand.log.appended")
	deadline := time.Now().Add(100 * time.Millisecond)
	for time.Now().Before(deadline) {
		message, ok := tryReadSocketMessage(ownerClient, 20*time.Millisecond)
		if !ok {
			continue
		}
		if message.Type == "hand.started" {
			t.Fatalf("duplicate delayed task started a second hand: %#v", message)
		}
	}
	updated, err := server.store.RoomByID(roomID)
	if err != nil {
		t.Fatal(err)
	}
	if payload.Hand.HandID == "" || updated.CurrentGameID != payload.Hand.HandID {
		t.Fatalf("room current game=%s broadcast hand=%s", updated.CurrentGameID, payload.Hand.HandID)
	}
}

func testLifecycleOwnerTransferOrder(t *testing.T) {
	server := testServer(t)
	ownerToken := registerUser(t, server, "matrix_owner", "矩阵房主")
	nextToken := registerUser(t, server, "matrix_next", "循环顺位")
	laterToken := registerUser(t, server, "matrix_later", "后续玩家")
	roomID, inviteCode := createLifecycleRoom(t, server, ownerToken, 4, 2)
	joinLifecycleRoom(t, server, nextToken, inviteCode)
	joinLifecycleRoom(t, server, laterToken, inviteCode)
	takeLifecycleSeat(t, server, ownerToken, roomID, 3)
	takeLifecycleSeat(t, server, nextToken, roomID, 1)
	takeLifecycleSeat(t, server, laterToken, roomID, 2)
	nextUserID := lifecycleUserID(t, server, nextToken)

	leave := httptest.NewRequest(http.MethodDelete, "/api/rooms/"+roomID+"/members/me", nil)
	leave.Header.Set("Authorization", "Bearer "+ownerToken)
	result := httptest.NewRecorder()
	server.Routes().ServeHTTP(result, leave)
	if result.Code != http.StatusOK {
		t.Fatalf("owner leave status=%d body=%s", result.Code, result.Body.String())
	}
	room, err := server.store.RoomByID(roomID)
	if err != nil {
		t.Fatal(err)
	}
	if room.OwnerUserID != nextUserID || room.Status != "waiting" {
		t.Fatalf("owner transfer room=%#v, want wrapped seat 1 user %s", room, nextUserID)
	}
	for _, member := range room.Members {
		if member.UserID == nextUserID && member.Role != "owner" {
			t.Fatalf("transferred member role=%s, want owner", member.Role)
		}
	}
}

func testLifecycleOwnerLeaveClosesRoom(t *testing.T) {
	server := testServer(t)
	ownerToken := registerUser(t, server, "matrix_solo", "关闭房主")
	roomID, _ := createLifecycleRoom(t, server, ownerToken, 2, 2)
	takeLifecycleSeat(t, server, ownerToken, roomID, 1)

	leave := httptest.NewRequest(http.MethodDelete, "/api/rooms/"+roomID+"/members/me", nil)
	leave.Header.Set("Authorization", "Bearer "+ownerToken)
	result := httptest.NewRecorder()
	server.Routes().ServeHTTP(result, leave)
	if result.Code != http.StatusOK {
		t.Fatalf("solo owner leave status=%d body=%s", result.Code, result.Body.String())
	}
	room, err := server.store.RoomByID(roomID)
	if err != nil {
		t.Fatal(err)
	}
	if room.Status != "closed" || room.OwnerUserID != "" || room.CurrentGameID != "" {
		t.Fatalf("closed room=%#v, want no owner or current game", room)
	}

	start := httptest.NewRequest(http.MethodPost, "/api/rooms/"+roomID+"/start", nil)
	start.Header.Set("Authorization", "Bearer "+ownerToken)
	startResult := httptest.NewRecorder()
	server.Routes().ServeHTTP(startResult, start)
	if startResult.Code < http.StatusBadRequest {
		t.Fatalf("closed room start status=%d body=%s", startResult.Code, startResult.Body.String())
	}
}

func testLifecycleOwnerTransferBroadcastAndCommandAuthority(t *testing.T) {
	server := testServer(t)
	ownerToken := registerUser(t, server, "authority_owner", "权限旧房主")
	nextToken := registerUser(t, server, "authority_next", "权限新房主")
	observerToken := registerUser(t, server, "authority_observer", "权限观察者")
	roomID, inviteCode := createLifecycleRoom(t, server, ownerToken, 3, 2)
	joinLifecycleRoom(t, server, nextToken, inviteCode)
	joinLifecycleRoom(t, server, observerToken, inviteCode)
	takeLifecycleSeat(t, server, ownerToken, roomID, 3)
	takeLifecycleSeat(t, server, nextToken, roomID, 1)
	takeLifecycleSeat(t, server, observerToken, roomID, 2)
	nextUserID := lifecycleUserID(t, server, nextToken)

	ts := httptest.NewServer(server.Routes())
	defer ts.Close()
	ownerClient := dialSocket(t, ts.URL, "/api/socket?token="+ownerToken)
	defer ownerClient.Close()
	ownerMirrorClient := dialSocket(t, ts.URL, "/api/socket?token="+ownerToken)
	defer ownerMirrorClient.Close()
	nextClient := dialSocket(t, ts.URL, "/api/socket?token="+nextToken)
	defer nextClient.Close()
	observerClient := dialSocket(t, ts.URL, "/api/socket?token="+observerToken)
	defer observerClient.Close()
	subscribeSocket(t, ownerClient, roomID, "req_authority_owner_sub")
	subscribeSocket(t, ownerMirrorClient, roomID, "req_authority_owner_mirror_sub")
	subscribeSocket(t, nextClient, roomID, "req_authority_next_sub")
	subscribeSocket(t, observerClient, roomID, "req_authority_observer_sub")

	if err := writeClientTextFrame(ownerClient, []byte(`{"type":"room.leave","requestId":"req_authority_owner_leave","roomId":"`+roomID+`","payload":{}}`)); err != nil {
		t.Fatal(err)
	}
	leaveAck := readSocketMessageSkipping(t, ownerClient, "ack", nil)
	if leaveAck.RequestID != "req_authority_owner_leave" {
		t.Fatalf("owner leave ack requestId=%s, want req_authority_owner_leave", leaveAck.RequestID)
	}
	for _, client := range []*socketTestConn{nextClient, observerClient} {
		updated := readLifecycleRoomUpdate(t, client)
		if updated.OwnerUserID != nextUserID || updated.Status != "waiting" || len(updated.Members) != 2 {
			t.Fatalf("transferred room update=%#v, want new owner %s and two remaining members", updated, nextUserID)
		}
	}

	if err := writeClientTextFrame(ownerClient, []byte(`{"type":"room.start_hand","requestId":"req_old_owner_start","roomId":"`+roomID+`","payload":{}}`)); err != nil {
		t.Fatal(err)
	}
	oldOwnerError := readSocketMessageSkipping(t, ownerClient, "error", map[string]bool{
		"room.updated":      true,
		"hand.log.appended": true,
	})
	var errorPayload ErrorResponse
	if err := json.Unmarshal(oldOwnerError.Payload, &errorPayload); err != nil {
		t.Fatal(err)
	}
	if errorPayload.Code != "not_room_owner" {
		t.Fatalf("old owner start error=%s, want not_room_owner", errorPayload.Code)
	}

	if err := writeClientTextFrame(nextClient, []byte(`{"type":"room.start_hand","requestId":"req_new_owner_start","roomId":"`+roomID+`","payload":{}}`)); err != nil {
		t.Fatal(err)
	}
	ack := readSocketMessageSkipping(t, nextClient, "ack", map[string]bool{"hand.log.appended": true})
	if ack.RequestID != "req_new_owner_start" {
		t.Fatalf("new owner start ack requestId=%s, want req_new_owner_start", ack.RequestID)
	}
	readSocketMessage(t, nextClient, "hand.started")
	readSocketMessageSkipping(t, observerClient, "hand.started", map[string]bool{"hand.log.appended": true})
	for _, client := range []*socketTestConn{ownerClient, ownerMirrorClient} {
		assertNoSocketEventType(t, client, "hand.started", 100*time.Millisecond)
	}
}

func testLifecyclePlayerLeaveBroadcast(t *testing.T) {
	server := testServer(t)
	roomID, ownerToken, playerToken := setupRoomWithSeats(t, server)
	ownerUserID := lifecycleUserID(t, server, ownerToken)
	ts := httptest.NewServer(server.Routes())
	defer ts.Close()
	ownerClient := dialSocket(t, ts.URL, "/api/socket?token="+ownerToken)
	defer ownerClient.Close()
	playerClient := dialSocket(t, ts.URL, "/api/socket?token="+playerToken)
	defer playerClient.Close()
	subscribeSocket(t, ownerClient, roomID, "req_player_leave_owner_sub")
	subscribeSocket(t, playerClient, roomID, "req_player_leave_player_sub")

	leaveLifecycleRoom(t, server, playerToken, roomID)
	updated := readLifecycleRoomUpdate(t, ownerClient)
	if updated.OwnerUserID != ownerUserID || updated.Status != "waiting" || len(updated.Members) != 1 {
		t.Fatalf("player leave room update=%#v, want unchanged owner and one member", updated)
	}
	assertNoSocketEventType(t, playerClient, "room.updated", 100*time.Millisecond)
}

func testLifecycleRoomCloseBroadcast(t *testing.T) {
	server := testServer(t)
	ownerToken := registerUser(t, server, "close_broadcast_owner", "关闭广播房主")
	observerToken := registerUser(t, server, "close_broadcast_observer", "关闭广播成员")
	roomID, inviteCode := createLifecycleRoom(t, server, ownerToken, 2, 2)
	joinLifecycleRoom(t, server, observerToken, inviteCode)
	takeLifecycleSeat(t, server, ownerToken, roomID, 1)

	ts := httptest.NewServer(server.Routes())
	defer ts.Close()
	observerClient := dialSocket(t, ts.URL, "/api/socket?token="+observerToken)
	defer observerClient.Close()
	subscribeSocket(t, observerClient, roomID, "req_close_broadcast_observer_sub")

	leaveLifecycleRoom(t, server, ownerToken, roomID)
	updated := readLifecycleRoomUpdate(t, observerClient)
	if updated.Status != "closed" || updated.OwnerUserID != "" || updated.CurrentGameID != "" || len(updated.Members) != 1 {
		t.Fatalf("closed room update=%#v, want closed room without owner or current hand", updated)
	}
}

func readLifecycleRoomUpdate(t *testing.T, client *socketTestConn) storage.RoomRecord {
	t.Helper()
	message := readSocketMessageSkipping(t, client, "room.updated", map[string]bool{
		"hand.log.appended": true,
	})
	var payload struct {
		Room storage.RoomRecord `json:"room"`
	}
	if err := json.Unmarshal(message.Payload, &payload); err != nil {
		t.Fatal(err)
	}
	return payload.Room
}

func assertLifecycleEventOrder(eventTypes []string) error {
	want := []string{"hand.settled", "wallet.updated", "room.leaderboard.updated", "room.updated"}
	if len(eventTypes) != len(want) {
		return fmt.Errorf("lifecycle events=%v, want %v", eventTypes, want)
	}
	for index := range want {
		if eventTypes[index] != want[index] {
			return fmt.Errorf("lifecycle events=%v, want %v", eventTypes, want)
		}
	}
	return nil
}

type recentHandResultsFailStore struct {
	Store
}

func (s *recentHandResultsFailStore) RecentHandResultsByRoom(string, int) ([]storage.HandResultRecord, error) {
	return nil, errors.New("forced post-commit hand result reload failure")
}

type blockingFirstSaveStore struct {
	Store
	firstSaveStarted chan struct{}
	releaseFirstSave chan struct{}
	once             sync.Once
}

func newBlockingFirstSaveStore(store Store) *blockingFirstSaveStore {
	return &blockingFirstSaveStore{
		Store:            store,
		firstSaveStarted: make(chan struct{}),
		releaseFirstSave: make(chan struct{}),
	}
}

func (s *blockingFirstSaveStore) Save(gameState *game.Game) error {
	blocked := false
	s.once.Do(func() {
		blocked = true
		close(s.firstSaveStarted)
	})
	if blocked {
		<-s.releaseFirstSave
	}
	return s.Store.Save(gameState)
}

type authoritativeRoomOwnerStore struct {
	Store
	roomID string
	mu     sync.RWMutex
	owner  string
}

func (s *authoritativeRoomOwnerStore) RoomByID(roomID string) (*storage.RoomRecord, error) {
	room, err := s.Store.RoomByID(roomID)
	if err != nil || roomID != s.roomID {
		return room, err
	}
	s.mu.RLock()
	owner := s.owner
	s.mu.RUnlock()
	if owner != "" {
		room.OwnerUserID = owner
	}
	return room, nil
}

func (s *authoritativeRoomOwnerStore) setOwner(owner string) {
	s.mu.Lock()
	s.owner = owner
	s.mu.Unlock()
}

type roomReadMutationStore struct {
	Store
	roomID string
	mu     sync.Mutex
	armed  bool
	reads  int
	mutate func() error
}

func (s *roomReadMutationStore) RoomByID(roomID string) (*storage.RoomRecord, error) {
	room, err := s.Store.RoomByID(roomID)
	if err != nil || roomID != s.roomID {
		return room, err
	}
	s.mu.Lock()
	if !s.armed {
		s.mu.Unlock()
		return room, nil
	}
	s.reads++
	shouldMutate := s.reads == 2
	mutate := s.mutate
	s.mu.Unlock()
	if !shouldMutate || mutate == nil {
		return room, nil
	}
	if err := mutate(); err != nil {
		return nil, err
	}
	return s.Store.RoomByID(roomID)
}

func (s *roomReadMutationStore) arm(mutate func() error) {
	s.mu.Lock()
	s.armed = true
	s.reads = 0
	s.mutate = mutate
	s.mu.Unlock()
}

func createLifecycleRoom(t *testing.T, server *Server, ownerToken string, seatCount, minPlayers int) (string, string) {
	t.Helper()
	body := fmt.Sprintf(`{"ruleSetId":"long-holdem","seatCount":%d,"minPlayersToStart":%d}`, seatCount, minPlayers)
	request := httptest.NewRequest(http.MethodPost, "/api/rooms", bytes.NewBufferString(body))
	request.Header.Set("Authorization", "Bearer "+ownerToken)
	result := httptest.NewRecorder()
	server.Routes().ServeHTTP(result, request)
	if result.Code != http.StatusCreated {
		t.Fatalf("create room status=%d body=%s", result.Code, result.Body.String())
	}
	var room struct {
		ID         string `json:"id"`
		InviteCode string `json:"inviteCode"`
	}
	if err := json.Unmarshal(result.Body.Bytes(), &room); err != nil {
		t.Fatal(err)
	}
	return room.ID, room.InviteCode
}

func joinLifecycleRoom(t *testing.T, server *Server, token, inviteCode string) {
	t.Helper()
	request := httptest.NewRequest(http.MethodPost, "/api/rooms/join", bytes.NewBufferString(`{"inviteCode":"`+inviteCode+`"}`))
	request.Header.Set("Authorization", "Bearer "+token)
	result := httptest.NewRecorder()
	server.Routes().ServeHTTP(result, request)
	if result.Code != http.StatusOK {
		t.Fatalf("join room status=%d body=%s", result.Code, result.Body.String())
	}
}

func takeLifecycleSeat(t *testing.T, server *Server, token, roomID string, seatNo int) {
	t.Helper()
	request := httptest.NewRequest(http.MethodPost, "/api/rooms/"+roomID+"/seats/"+itoa(seatNo), bytes.NewBufferString(`{"buyInChips":1000}`))
	request.Header.Set("Authorization", "Bearer "+token)
	result := httptest.NewRecorder()
	server.Routes().ServeHTTP(result, request)
	if result.Code != http.StatusOK {
		t.Fatalf("take seat status=%d body=%s", result.Code, result.Body.String())
	}
}

func leaveLifecycleRoom(t *testing.T, server *Server, token, roomID string) {
	t.Helper()
	request := httptest.NewRequest(http.MethodDelete, "/api/rooms/"+roomID+"/members/me", nil)
	request.Header.Set("Authorization", "Bearer "+token)
	result := httptest.NewRecorder()
	server.Routes().ServeHTTP(result, request)
	if result.Code != http.StatusOK {
		t.Fatalf("leave room status=%d body=%s", result.Code, result.Body.String())
	}
}

func assertNoSocketEventType(t *testing.T, client *socketTestConn, eventType string, timeout time.Duration) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		message, ok := tryReadSocketMessage(client, 20*time.Millisecond)
		if !ok {
			continue
		}
		if message.Type == eventType {
			t.Fatalf("unexpected %s event: %#v", eventType, message)
		}
	}
}

func lifecycleUserID(t *testing.T, server *Server, token string) string {
	t.Helper()
	request := httptest.NewRequest(http.MethodGet, "/api/me", nil)
	request.Header.Set("Authorization", "Bearer "+token)
	result := httptest.NewRecorder()
	server.Routes().ServeHTTP(result, request)
	if result.Code != http.StatusOK {
		t.Fatalf("profile status=%d body=%s", result.Code, result.Body.String())
	}
	var profile struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(result.Body.Bytes(), &profile); err != nil {
		t.Fatal(err)
	}
	return profile.ID
}
