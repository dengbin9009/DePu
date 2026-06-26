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
				UserID   string `json:"userId"`
				Nickname string `json:"nickname"`
				Profit   int    `json:"profit"`
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
