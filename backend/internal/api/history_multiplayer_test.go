package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRoomRecentHandsAndUserHands(t *testing.T) {
	server := testServer(t)
	roomID, ownerToken, playerToken := setupRoomWithSeats(t, server)

	startReq := httptest.NewRequest(http.MethodPost, "/api/rooms/"+roomID+"/start", nil)
	startReq.Header.Set("Authorization", "Bearer "+ownerToken)
	startRes := httptest.NewRecorder()
	server.Routes().ServeHTTP(startRes, startReq)
	if startRes.Code != http.StatusOK {
		t.Fatalf("start status=%d body=%s", startRes.Code, startRes.Body.String())
	}

	for i := 0; i < 8; i++ {
		currentReq := httptest.NewRequest(http.MethodGet, "/api/rooms/"+roomID+"/current-hand", nil)
		currentReq.Header.Set("Authorization", "Bearer "+ownerToken)
		currentRes := httptest.NewRecorder()
		server.Routes().ServeHTTP(currentRes, currentReq)
		if currentRes.Code != http.StatusOK {
			t.Fatalf("current hand status=%d body=%s", currentRes.Code, currentRes.Body.String())
		}
		var hand map[string]any
		if err := json.Unmarshal(currentRes.Body.Bytes(), &hand); err != nil {
			t.Fatal(err)
		}
		if hand["status"] == "finished" {
			break
		}
		currentSeat, _ := hand["currentSeat"].(float64)
		token := playerToken
		if int(currentSeat) == 1 {
			token = ownerToken
		}
		action := "check"
		if available, ok := hand["availableActions"].([]any); ok {
			has := map[string]bool{}
			for _, item := range available {
				if text, ok := item.(string); ok {
					has[text] = true
				}
			}
			switch {
			case has["check"]:
				action = "check"
			case has["call"]:
				action = "call"
			case has["fold"]:
				action = "fold"
			default:
				action = "all_in"
			}
		}
		actReq := httptest.NewRequest(http.MethodPost, "/api/rooms/"+roomID+"/actions", bytes.NewReader([]byte(`{"action":"`+action+`","amount":0}`)))
		actReq.Header.Set("Authorization", "Bearer "+token)
		actRes := httptest.NewRecorder()
		server.Routes().ServeHTTP(actRes, actReq)
		if actRes.Code != http.StatusOK {
			t.Fatalf("action status=%d body=%s", actRes.Code, actRes.Body.String())
		}
	}
	finalReq := httptest.NewRequest(http.MethodGet, "/api/rooms/"+roomID+"/current-hand", nil)
	finalReq.Header.Set("Authorization", "Bearer "+ownerToken)
	finalRes := httptest.NewRecorder()
	server.Routes().ServeHTTP(finalRes, finalReq)
	if finalRes.Code == http.StatusOK {
		var finalState map[string]any
		if err := json.Unmarshal(finalRes.Body.Bytes(), &finalState); err == nil {
			if finalState["status"] != "finished" {
				t.Fatalf("hand did not finish, status=%v", finalState["status"])
			}
		}
	}

	handsReq := httptest.NewRequest(http.MethodGet, "/api/rooms/"+roomID+"/hands/recent", nil)
	handsReq.Header.Set("Authorization", "Bearer "+ownerToken)
	handsRes := httptest.NewRecorder()
	server.Routes().ServeHTTP(handsRes, handsReq)
	if handsRes.Code != http.StatusOK {
		t.Fatalf("recent hands status=%d body=%s", handsRes.Code, handsRes.Body.String())
	}
	var handsPayload struct {
		Items []struct {
			HandID        string `json:"handId"`
			HandNo        int    `json:"handNo"`
			WinnerSummary string `json:"winnerSummary"`
			PotSummary    string `json:"potSummary"`
			Participants  []struct {
				UserID        string `json:"userId"`
				Nickname      string `json:"nickname"`
				Profit        int    `json:"profit"`
				AwardAmount   int    `json:"awardAmount"`
				HandCommitted int    `json:"handCommitted"`
			} `json:"participants"`
		} `json:"items"`
	}
	if err := json.Unmarshal(handsRes.Body.Bytes(), &handsPayload); err != nil {
		t.Fatal(err)
	}
	if len(handsPayload.Items) == 0 {
		t.Fatal("expected archived room hand")
	}
	if handsPayload.Items[0].HandID == "" || handsPayload.Items[0].HandNo == 0 {
		t.Fatal("expected hand id and hand no")
	}
	if len(handsPayload.Items[0].Participants) != 2 {
		t.Fatalf("participants=%d want 2", len(handsPayload.Items[0].Participants))
	}
	for _, participant := range handsPayload.Items[0].Participants {
		if participant.HandCommitted <= 0 {
			t.Fatalf("expected handCommitted > 0, got=%d", participant.HandCommitted)
		}
		if participant.AwardAmount < 0 {
			t.Fatalf("expected non-negative awardAmount, got=%d", participant.AwardAmount)
		}
	}

	myReq := httptest.NewRequest(http.MethodGet, "/api/me/hands", nil)
	myReq.Header.Set("Authorization", "Bearer "+ownerToken)
	myRes := httptest.NewRecorder()
	server.Routes().ServeHTTP(myRes, myReq)
	if myRes.Code != http.StatusOK {
		t.Fatalf("my hands status=%d body=%s", myRes.Code, myRes.Body.String())
	}
	var myPayload struct {
		Items []struct {
			HandID        string `json:"handId"`
			RoomID        string `json:"roomId"`
			Nickname      string `json:"nickname"`
			Profit        int    `json:"profit"`
			WinnerSummary string `json:"winnerSummary"`
		} `json:"items"`
	}
	if err := json.Unmarshal(myRes.Body.Bytes(), &myPayload); err != nil {
		t.Fatal(err)
	}
	if len(myPayload.Items) == 0 {
		t.Fatal("expected personal hand history")
	}
	if myPayload.Items[0].RoomID != roomID {
		t.Fatalf("roomID=%s want %s", myPayload.Items[0].RoomID, roomID)
	}
}

func TestRoomLeaderboardAndFormalReplay(t *testing.T) {
	server := testServer(t)
	roomID, ownerToken, playerToken := setupRoomWithSeats(t, server)
	playRoomHandToSettlement(t, server, roomID, ownerToken, playerToken)

	leaderboardReq := httptest.NewRequest(http.MethodGet, "/api/rooms/"+roomID+"/leaderboard", nil)
	leaderboardReq.Header.Set("Authorization", "Bearer "+ownerToken)
	leaderboardRes := httptest.NewRecorder()
	server.Routes().ServeHTTP(leaderboardRes, leaderboardReq)
	if leaderboardRes.Code != http.StatusOK {
		t.Fatalf("leaderboard status=%d body=%s", leaderboardRes.Code, leaderboardRes.Body.String())
	}
	var leaderboardPayload struct {
		Items []struct {
			UserID        string `json:"userId"`
			Nickname      string `json:"nickname"`
			HandsPlayed   int    `json:"handsPlayed"`
			HandsWon      int    `json:"handsWon"`
			NetProfit     int    `json:"netProfit"`
			BiggestPotWon int    `json:"biggestPotWon"`
			LastSettledAt string `json:"lastSettledAt"`
		} `json:"items"`
	}
	if err := json.Unmarshal(leaderboardRes.Body.Bytes(), &leaderboardPayload); err != nil {
		t.Fatal(err)
	}
	if len(leaderboardPayload.Items) != 2 {
		t.Fatalf("leaderboard items=%d want 2", len(leaderboardPayload.Items))
	}
	for _, item := range leaderboardPayload.Items {
		if item.UserID == "" || item.Nickname == "" || item.HandsPlayed == 0 || item.LastSettledAt == "" {
			t.Fatalf("leaderboard item incomplete: %#v", item)
		}
	}

	handsReq := httptest.NewRequest(http.MethodGet, "/api/rooms/"+roomID+"/hands/recent", nil)
	handsReq.Header.Set("Authorization", "Bearer "+ownerToken)
	handsRes := httptest.NewRecorder()
	server.Routes().ServeHTTP(handsRes, handsReq)
	var handsPayload struct {
		Items []struct {
			HandID string `json:"handId"`
		} `json:"items"`
	}
	if err := json.Unmarshal(handsRes.Body.Bytes(), &handsPayload); err != nil {
		t.Fatal(err)
	}
	if len(handsPayload.Items) == 0 || handsPayload.Items[0].HandID == "" {
		t.Fatalf("missing archived hand: %#v", handsPayload)
	}

	replayReq := httptest.NewRequest(http.MethodGet, "/api/rooms/"+roomID+"/hands/"+handsPayload.Items[0].HandID+"/replay", nil)
	replayReq.Header.Set("Authorization", "Bearer "+ownerToken)
	replayRes := httptest.NewRecorder()
	server.Routes().ServeHTTP(replayRes, replayReq)
	if replayRes.Code != http.StatusOK {
		t.Fatalf("replay status=%d body=%s", replayRes.Code, replayRes.Body.String())
	}
	var replayPayload struct {
		HandID string `json:"handId"`
		Steps  []struct {
			Seq         int      `json:"seq"`
			Stage       string   `json:"stage"`
			CurrentSeat int      `json:"currentSeat"`
			BoardCards  []string `json:"boardCards"`
			Action      *struct {
				Type   string `json:"type"`
				SeatNo int    `json:"seatNo"`
			} `json:"action,omitempty"`
			Players []struct {
				SeatNo    int      `json:"seatNo"`
				HoleCards []string `json:"holeCards,omitempty"`
				Status    string   `json:"status"`
			} `json:"players"`
		} `json:"steps"`
	}
	if err := json.Unmarshal(replayRes.Body.Bytes(), &replayPayload); err != nil {
		t.Fatal(err)
	}
	if replayPayload.HandID != handsPayload.Items[0].HandID || len(replayPayload.Steps) < 2 {
		t.Fatalf("replay payload incomplete: %#v", replayPayload)
	}
	for _, step := range replayPayload.Steps {
		for _, player := range step.Players {
			if step.Stage != "showdown" && len(player.HoleCards) != 0 {
				t.Fatalf("replay leaked hidden hole cards before showdown: step=%#v player=%#v", step, player)
			}
		}
	}
}

func TestFormalReplayKeepsHoleCardsPrivateWhenHandSettlesByFold(t *testing.T) {
	server := testServer(t)
	roomID, ownerToken, playerToken := setupRoomWithSeats(t, server)

	startReq := httptest.NewRequest(http.MethodPost, "/api/rooms/"+roomID+"/start", nil)
	startReq.Header.Set("Authorization", "Bearer "+ownerToken)
	startRes := httptest.NewRecorder()
	server.Routes().ServeHTTP(startRes, startReq)
	if startRes.Code != http.StatusOK {
		t.Fatalf("start status=%d body=%s", startRes.Code, startRes.Body.String())
	}

	currentReq := httptest.NewRequest(http.MethodGet, "/api/rooms/"+roomID+"/current-hand", nil)
	currentReq.Header.Set("Authorization", "Bearer "+ownerToken)
	currentRes := httptest.NewRecorder()
	server.Routes().ServeHTTP(currentRes, currentReq)
	if currentRes.Code != http.StatusOK {
		t.Fatalf("current hand status=%d body=%s", currentRes.Code, currentRes.Body.String())
	}
	var currentHand struct {
		CurrentSeat int `json:"currentSeat"`
	}
	if err := json.Unmarshal(currentRes.Body.Bytes(), &currentHand); err != nil {
		t.Fatal(err)
	}
	actingToken := playerToken
	if currentHand.CurrentSeat == 1 {
		actingToken = ownerToken
	}

	foldReq := httptest.NewRequest(http.MethodPost, "/api/rooms/"+roomID+"/actions", bytes.NewReader([]byte(`{"action":"fold","amount":0}`)))
	foldReq.Header.Set("Authorization", "Bearer "+actingToken)
	foldRes := httptest.NewRecorder()
	server.Routes().ServeHTTP(foldRes, foldReq)
	if foldRes.Code != http.StatusOK {
		t.Fatalf("fold status=%d body=%s", foldRes.Code, foldRes.Body.String())
	}

	handsReq := httptest.NewRequest(http.MethodGet, "/api/rooms/"+roomID+"/hands/recent", nil)
	handsReq.Header.Set("Authorization", "Bearer "+ownerToken)
	handsRes := httptest.NewRecorder()
	server.Routes().ServeHTTP(handsRes, handsReq)
	if handsRes.Code != http.StatusOK {
		t.Fatalf("recent hands status=%d body=%s", handsRes.Code, handsRes.Body.String())
	}
	var handsPayload struct {
		Items []struct {
			HandID string `json:"handId"`
		} `json:"items"`
	}
	if err := json.Unmarshal(handsRes.Body.Bytes(), &handsPayload); err != nil {
		t.Fatal(err)
	}
	if len(handsPayload.Items) == 0 {
		t.Fatal("expected archived folded hand")
	}

	replayReq := httptest.NewRequest(http.MethodGet, "/api/rooms/"+roomID+"/hands/"+handsPayload.Items[0].HandID+"/replay", nil)
	replayReq.Header.Set("Authorization", "Bearer "+ownerToken)
	replayRes := httptest.NewRecorder()
	server.Routes().ServeHTTP(replayRes, replayReq)
	if replayRes.Code != http.StatusOK {
		t.Fatalf("replay status=%d body=%s", replayRes.Code, replayRes.Body.String())
	}
	var replayPayload struct {
		Steps []struct {
			Stage   string `json:"stage"`
			Players []struct {
				SeatNo    int      `json:"seatNo"`
				HoleCards []string `json:"holeCards,omitempty"`
			} `json:"players"`
		} `json:"steps"`
	}
	if err := json.Unmarshal(replayRes.Body.Bytes(), &replayPayload); err != nil {
		t.Fatal(err)
	}
	if len(replayPayload.Steps) == 0 || replayPayload.Steps[len(replayPayload.Steps)-1].Stage != "finished" {
		t.Fatalf("expected finished replay step: %#v", replayPayload.Steps)
	}
	for _, step := range replayPayload.Steps {
		for _, player := range step.Players {
			if len(player.HoleCards) != 0 {
				t.Fatalf("folded hand replay leaked private hole cards: step=%#v player=%#v", step, player)
			}
		}
	}
}

func TestFormalReplayPreservesHistoricalStepsAcrossMultipleSaves(t *testing.T) {
	server := testServer(t)
	roomID, ownerToken, playerToken := setupRoomWithSeats(t, server)

	startReq := httptest.NewRequest(http.MethodPost, "/api/rooms/"+roomID+"/start", nil)
	startReq.Header.Set("Authorization", "Bearer "+ownerToken)
	startRes := httptest.NewRecorder()
	server.Routes().ServeHTTP(startRes, startReq)
	if startRes.Code != http.StatusOK {
		t.Fatalf("start status=%d body=%s", startRes.Code, startRes.Body.String())
	}

	for actionIndex := 0; actionIndex < 2; actionIndex++ {
		currentReq := httptest.NewRequest(http.MethodGet, "/api/rooms/"+roomID+"/current-hand", nil)
		currentReq.Header.Set("Authorization", "Bearer "+ownerToken)
		currentRes := httptest.NewRecorder()
		server.Routes().ServeHTTP(currentRes, currentReq)
		if currentRes.Code != http.StatusOK {
			t.Fatalf("current hand status=%d body=%s", currentRes.Code, currentRes.Body.String())
		}
		var currentHand struct {
			CurrentSeat int `json:"currentSeat"`
		}
		if err := json.Unmarshal(currentRes.Body.Bytes(), &currentHand); err != nil {
			t.Fatal(err)
		}
		actingToken := playerToken
		if currentHand.CurrentSeat == 1 {
			actingToken = ownerToken
		}
		actionReq := httptest.NewRequest(http.MethodPost, "/api/rooms/"+roomID+"/actions", bytes.NewReader([]byte(`{"action":"all_in","amount":0}`)))
		actionReq.Header.Set("Authorization", "Bearer "+actingToken)
		actionRes := httptest.NewRecorder()
		server.Routes().ServeHTTP(actionRes, actionReq)
		if actionRes.Code != http.StatusOK {
			t.Fatalf("all-in %d status=%d body=%s", actionIndex+1, actionRes.Code, actionRes.Body.String())
		}
	}

	handsReq := httptest.NewRequest(http.MethodGet, "/api/rooms/"+roomID+"/hands/recent", nil)
	handsReq.Header.Set("Authorization", "Bearer "+ownerToken)
	handsRes := httptest.NewRecorder()
	server.Routes().ServeHTTP(handsRes, handsReq)
	if handsRes.Code != http.StatusOK {
		t.Fatalf("recent hands status=%d body=%s", handsRes.Code, handsRes.Body.String())
	}
	var handsPayload struct {
		Items []struct {
			HandID string `json:"handId"`
		} `json:"items"`
	}
	if err := json.Unmarshal(handsRes.Body.Bytes(), &handsPayload); err != nil {
		t.Fatal(err)
	}
	if len(handsPayload.Items) != 1 {
		t.Fatalf("hands=%d want 1", len(handsPayload.Items))
	}

	replayReq := httptest.NewRequest(http.MethodGet, "/api/rooms/"+roomID+"/hands/"+handsPayload.Items[0].HandID+"/replay", nil)
	replayReq.Header.Set("Authorization", "Bearer "+ownerToken)
	replayRes := httptest.NewRecorder()
	server.Routes().ServeHTTP(replayRes, replayReq)
	if replayRes.Code != http.StatusOK {
		t.Fatalf("replay status=%d body=%s", replayRes.Code, replayRes.Body.String())
	}
	var replayPayload struct {
		Steps []struct {
			Seq        int      `json:"seq"`
			Stage      string   `json:"stage"`
			BoardCards []string `json:"boardCards"`
			Players    []struct {
				HoleCards []string `json:"holeCards,omitempty"`
			} `json:"players"`
		} `json:"steps"`
	}
	if err := json.Unmarshal(replayRes.Body.Bytes(), &replayPayload); err != nil {
		t.Fatal(err)
	}
	if len(replayPayload.Steps) < 4 {
		t.Fatalf("replay steps=%d want at least 4", len(replayPayload.Steps))
	}
	firstActionStep := replayPayload.Steps[1]
	if firstActionStep.Stage != "preflop" || len(firstActionStep.BoardCards) != 0 {
		t.Fatalf("first action step was overwritten by later state: %#v", firstActionStep)
	}
	for _, player := range firstActionStep.Players {
		if len(player.HoleCards) != 0 {
			t.Fatalf("first action step leaked hole cards: %#v", firstActionStep)
		}
	}
	finalStep := replayPayload.Steps[len(replayPayload.Steps)-1]
	if finalStep.Stage != "showdown" || len(finalStep.BoardCards) != 5 {
		t.Fatalf("final replay step=%#v want showdown with five board cards", finalStep)
	}
	for _, player := range finalStep.Players {
		if len(player.HoleCards) != 2 {
			t.Fatalf("final replay step should reveal both hole cards: %#v", finalStep)
		}
	}
}

func TestRoomLeaderboardAndReplayRejectNonMembers(t *testing.T) {
	server := testServer(t)
	roomID, ownerToken, playerToken := setupRoomWithSeats(t, server)
	outsiderToken := registerUser(t, server, "history_outsider", "历史旁观")
	playRoomHandToSettlement(t, server, roomID, ownerToken, playerToken)

	leaderboardReq := httptest.NewRequest(http.MethodGet, "/api/rooms/"+roomID+"/leaderboard", nil)
	leaderboardReq.Header.Set("Authorization", "Bearer "+outsiderToken)
	leaderboardRes := httptest.NewRecorder()
	server.Routes().ServeHTTP(leaderboardRes, leaderboardReq)
	if leaderboardRes.Code != http.StatusForbidden && leaderboardRes.Code != http.StatusNotFound {
		t.Fatalf("leaderboard outsider status=%d body=%s", leaderboardRes.Code, leaderboardRes.Body.String())
	}

	outsiderHandsReq := httptest.NewRequest(http.MethodGet, "/api/rooms/"+roomID+"/hands/recent", nil)
	outsiderHandsReq.Header.Set("Authorization", "Bearer "+outsiderToken)
	outsiderHandsRes := httptest.NewRecorder()
	server.Routes().ServeHTTP(outsiderHandsRes, outsiderHandsReq)
	if outsiderHandsRes.Code != http.StatusForbidden {
		t.Fatalf("recent hands outsider status=%d body=%s", outsiderHandsRes.Code, outsiderHandsRes.Body.String())
	}

	handsReq := httptest.NewRequest(http.MethodGet, "/api/rooms/"+roomID+"/hands/recent", nil)
	handsReq.Header.Set("Authorization", "Bearer "+ownerToken)
	handsRes := httptest.NewRecorder()
	server.Routes().ServeHTTP(handsRes, handsReq)
	var handsPayload struct {
		Items []struct {
			HandID string `json:"handId"`
		} `json:"items"`
	}
	if err := json.Unmarshal(handsRes.Body.Bytes(), &handsPayload); err != nil {
		t.Fatal(err)
	}
	replayReq := httptest.NewRequest(http.MethodGet, "/api/rooms/"+roomID+"/hands/"+handsPayload.Items[0].HandID+"/replay", nil)
	replayReq.Header.Set("Authorization", "Bearer "+outsiderToken)
	replayRes := httptest.NewRecorder()
	server.Routes().ServeHTTP(replayRes, replayReq)
	if replayRes.Code != http.StatusForbidden {
		t.Fatalf("replay outsider status=%d body=%s", replayRes.Code, replayRes.Body.String())
	}
	var replayError ErrorResponse
	if err := json.Unmarshal(replayRes.Body.Bytes(), &replayError); err != nil {
		t.Fatal(err)
	}
	if replayError.Code != "forbidden" {
		t.Fatalf("replay outsider code=%s body=%s", replayError.Code, replayRes.Body.String())
	}
}

func TestRecentRoomHandsDoNotExposeArchivedHoleCards(t *testing.T) {
	server := testServer(t)
	roomID, ownerToken, playerToken := setupRoomWithSeats(t, server)
	playRoomHandToSettlement(t, server, roomID, ownerToken, playerToken)

	request := httptest.NewRequest(http.MethodGet, "/api/rooms/"+roomID+"/hands/recent", nil)
	request.Header.Set("Authorization", "Bearer "+ownerToken)
	response := httptest.NewRecorder()
	server.Routes().ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("recent hands status=%d body=%s", response.Code, response.Body.String())
	}
	var payload struct {
		Items []struct {
			Participants []struct {
				HoleCards []string `json:"holeCards"`
			} `json:"participants"`
		} `json:"items"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}
	if len(payload.Items) == 0 {
		t.Fatal("expected recent hand summary")
	}
	for _, participant := range payload.Items[0].Participants {
		if len(participant.HoleCards) != 0 {
			t.Fatalf("recent hand summary leaked archived hole cards=%v", participant.HoleCards)
		}
	}
}

func TestCurrentRoomHandRequiresMembershipAndHidesOtherPlayersHoleCards(t *testing.T) {
	server := testServer(t)
	roomID, ownerToken, playerToken := setupRoomWithSeats(t, server)
	outsiderToken := registerUser(t, server, "current_hand_outsider", "当前手牌旁观者")

	startReq := httptest.NewRequest(http.MethodPost, "/api/rooms/"+roomID+"/start", nil)
	startReq.Header.Set("Authorization", "Bearer "+ownerToken)
	startRes := httptest.NewRecorder()
	server.Routes().ServeHTTP(startRes, startReq)
	if startRes.Code != http.StatusOK {
		t.Fatalf("start status=%d body=%s", startRes.Code, startRes.Body.String())
	}

	outsiderReq := httptest.NewRequest(http.MethodGet, "/api/rooms/"+roomID+"/current-hand", nil)
	outsiderReq.Header.Set("Authorization", "Bearer "+outsiderToken)
	outsiderRes := httptest.NewRecorder()
	server.Routes().ServeHTTP(outsiderRes, outsiderReq)
	if outsiderRes.Code != http.StatusForbidden {
		t.Fatalf("current hand outsider status=%d body=%s", outsiderRes.Code, outsiderRes.Body.String())
	}

	assertCurrentHandHoleCardView(t, server, roomID, ownerToken, 1)
	assertCurrentHandHoleCardView(t, server, roomID, playerToken, 2)
}

func assertCurrentHandHoleCardView(t *testing.T, server *Server, roomID, token string, visibleSeatNo int) {
	t.Helper()
	request := httptest.NewRequest(http.MethodGet, "/api/rooms/"+roomID+"/current-hand", nil)
	request.Header.Set("Authorization", "Bearer "+token)
	response := httptest.NewRecorder()
	server.Routes().ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("current hand status=%d body=%s", response.Code, response.Body.String())
	}
	var payload struct {
		Players []struct {
			SeatNo    int      `json:"seatNo"`
			HoleCards []string `json:"holeCards"`
		} `json:"players"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}
	for _, player := range payload.Players {
		if player.SeatNo == visibleSeatNo {
			if len(player.HoleCards) != 2 {
				t.Fatalf("seat %d hole cards=%v, want own cards", player.SeatNo, player.HoleCards)
			}
			continue
		}
		if len(player.HoleCards) != 0 {
			t.Fatalf("seat %d leaked hole cards=%v to seat %d", player.SeatNo, player.HoleCards, visibleSeatNo)
		}
	}
}

func TestSettlementKeepsWalletHistoryAndProfileConsistent(t *testing.T) {
	server := testServer(t)
	roomID, ownerToken, playerToken := setupRoomWithSeats(t, server)

	startReq := httptest.NewRequest(http.MethodPost, "/api/rooms/"+roomID+"/start", nil)
	startReq.Header.Set("Authorization", "Bearer "+ownerToken)
	startRes := httptest.NewRecorder()
	server.Routes().ServeHTTP(startRes, startReq)
	if startRes.Code != http.StatusOK {
		t.Fatalf("start status=%d body=%s", startRes.Code, startRes.Body.String())
	}

	for i := 0; i < 8; i++ {
		currentReq := httptest.NewRequest(http.MethodGet, "/api/rooms/"+roomID+"/current-hand", nil)
		currentReq.Header.Set("Authorization", "Bearer "+ownerToken)
		currentRes := httptest.NewRecorder()
		server.Routes().ServeHTTP(currentRes, currentReq)
		var hand map[string]any
		_ = json.Unmarshal(currentRes.Body.Bytes(), &hand)
		if hand["status"] == "finished" {
			break
		}
		currentSeat, _ := hand["currentSeat"].(float64)
		token := playerToken
		if int(currentSeat) == 1 {
			token = ownerToken
		}
		action := "check"
		if available, ok := hand["availableActions"].([]any); ok {
			has := map[string]bool{}
			for _, item := range available {
				if text, ok := item.(string); ok {
					has[text] = true
				}
			}
			switch {
			case has["check"]:
				action = "check"
			case has["call"]:
				action = "call"
			case has["fold"]:
				action = "fold"
			default:
				action = "all_in"
			}
		}
		actReq := httptest.NewRequest(http.MethodPost, "/api/rooms/"+roomID+"/actions", bytes.NewReader([]byte(`{"action":"`+action+`","amount":0}`)))
		actReq.Header.Set("Authorization", "Bearer "+token)
		actRes := httptest.NewRecorder()
		server.Routes().ServeHTTP(actRes, actReq)
		if actRes.Code != http.StatusOK {
			t.Fatalf("action status=%d body=%s", actRes.Code, actRes.Body.String())
		}
	}

	ownerWalletReq := httptest.NewRequest(http.MethodGet, "/api/me/wallet", nil)
	ownerWalletReq.Header.Set("Authorization", "Bearer "+ownerToken)
	ownerWalletRes := httptest.NewRecorder()
	server.Routes().ServeHTTP(ownerWalletRes, ownerWalletReq)
	if ownerWalletRes.Code != http.StatusOK {
		t.Fatalf("owner wallet status=%d body=%s", ownerWalletRes.Code, ownerWalletRes.Body.String())
	}
	var ownerWallet struct {
		Balance      int `json:"balance"`
		Transactions []struct {
			Type string `json:"type"`
		} `json:"transactions"`
	}
	if err := json.Unmarshal(ownerWalletRes.Body.Bytes(), &ownerWallet); err != nil {
		t.Fatal(err)
	}
	if len(ownerWallet.Transactions) < 2 {
		t.Fatalf("expected at least buy-in and settlement transactions, got=%d", len(ownerWallet.Transactions))
	}

	ownerMeReq := httptest.NewRequest(http.MethodGet, "/api/me", nil)
	ownerMeReq.Header.Set("Authorization", "Bearer "+ownerToken)
	ownerMeRes := httptest.NewRecorder()
	server.Routes().ServeHTTP(ownerMeRes, ownerMeReq)
	if ownerMeRes.Code != http.StatusOK {
		t.Fatalf("owner me status=%d body=%s", ownerMeRes.Code, ownerMeRes.Body.String())
	}
	var ownerProfile struct {
		HandsPlayed  int    `json:"handsPlayed"`
		TotalProfit  int    `json:"totalProfit"`
		LastPlayedAt string `json:"lastPlayedAt"`
	}
	if err := json.Unmarshal(ownerMeRes.Body.Bytes(), &ownerProfile); err != nil {
		t.Fatal(err)
	}
	if ownerProfile.HandsPlayed < 1 {
		t.Fatalf("expected handsPlayed >= 1, got=%d", ownerProfile.HandsPlayed)
	}
	if ownerProfile.LastPlayedAt == "" {
		t.Fatal("expected lastPlayedAt to be set")
	}

	handsReq := httptest.NewRequest(http.MethodGet, "/api/me/hands", nil)
	handsReq.Header.Set("Authorization", "Bearer "+ownerToken)
	handsRes := httptest.NewRecorder()
	server.Routes().ServeHTTP(handsRes, handsReq)
	if handsRes.Code != http.StatusOK {
		t.Fatalf("owner hands status=%d body=%s", handsRes.Code, handsRes.Body.String())
	}
	var handsPayload struct {
		Items []struct {
			Profit int `json:"profit"`
		} `json:"items"`
	}
	if err := json.Unmarshal(handsRes.Body.Bytes(), &handsPayload); err != nil {
		t.Fatal(err)
	}
	if len(handsPayload.Items) == 0 {
		t.Fatal("expected at least one history item")
	}
	if ownerProfile.TotalProfit != handsPayload.Items[0].Profit {
		t.Fatalf("totalProfit=%d want latestHandProfit=%d", ownerProfile.TotalProfit, handsPayload.Items[0].Profit)
	}
}

func playRoomHandToSettlement(t *testing.T, server *Server, roomID, ownerToken, playerToken string) {
	t.Helper()
	startReq := httptest.NewRequest(http.MethodPost, "/api/rooms/"+roomID+"/start", nil)
	startReq.Header.Set("Authorization", "Bearer "+ownerToken)
	startRes := httptest.NewRecorder()
	server.Routes().ServeHTTP(startRes, startReq)
	if startRes.Code != http.StatusOK {
		t.Fatalf("start status=%d body=%s", startRes.Code, startRes.Body.String())
	}

	for i := 0; i < 12; i++ {
		currentReq := httptest.NewRequest(http.MethodGet, "/api/rooms/"+roomID+"/current-hand", nil)
		currentReq.Header.Set("Authorization", "Bearer "+ownerToken)
		currentRes := httptest.NewRecorder()
		server.Routes().ServeHTTP(currentRes, currentReq)
		if currentRes.Code == http.StatusNotFound {
			return
		}
		if currentRes.Code != http.StatusOK {
			t.Fatalf("current hand status=%d body=%s", currentRes.Code, currentRes.Body.String())
		}
		var hand map[string]any
		if err := json.Unmarshal(currentRes.Body.Bytes(), &hand); err != nil {
			t.Fatal(err)
		}
		if hand["status"] == "finished" {
			return
		}
		currentSeat, _ := hand["currentSeat"].(float64)
		token := playerToken
		if int(currentSeat) == 1 {
			token = ownerToken
		}
		action := "check"
		if available, ok := hand["availableActions"].([]any); ok {
			has := map[string]bool{}
			for _, item := range available {
				if text, ok := item.(string); ok {
					has[text] = true
				}
			}
			switch {
			case has["check"]:
				action = "check"
			case has["call"]:
				action = "call"
			case has["fold"]:
				action = "fold"
			default:
				action = "all_in"
			}
		}
		actReq := httptest.NewRequest(http.MethodPost, "/api/rooms/"+roomID+"/actions", bytes.NewReader([]byte(`{"action":"`+action+`","amount":0}`)))
		actReq.Header.Set("Authorization", "Bearer "+token)
		actRes := httptest.NewRecorder()
		server.Routes().ServeHTTP(actRes, actReq)
		if actRes.Code != http.StatusOK {
			t.Fatalf("action status=%d body=%s", actRes.Code, actRes.Body.String())
		}
	}
	t.Fatal("hand did not settle within expected actions")
}
