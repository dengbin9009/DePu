package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRegisterLoginAndMe(t *testing.T) {
	server := testServer(t)
	registerBody := []byte(`{"username":"alice","password":"password1","nickname":"Alice"}`)
	registerRes := httptest.NewRecorder()
	server.Routes().ServeHTTP(registerRes, httptest.NewRequest(http.MethodPost, "/api/auth/register", bytes.NewReader(registerBody)))
	if registerRes.Code != http.StatusCreated {
		t.Fatalf("register status = %d body=%s", registerRes.Code, registerRes.Body.String())
	}
	var registered map[string]any
	if err := json.Unmarshal(registerRes.Body.Bytes(), &registered); err != nil { t.Fatal(err) }
	token, _ := registered["token"].(string)
	if token == "" { t.Fatal("expected token") }

	meReq := httptest.NewRequest(http.MethodGet, "/api/me", nil)
	meReq.Header.Set("Authorization", "Bearer "+token)
	meRes := httptest.NewRecorder()
	server.Routes().ServeHTTP(meRes, meReq)
	if meRes.Code != http.StatusOK {
		t.Fatalf("me status = %d body=%s", meRes.Code, meRes.Body.String())
	}

	loginBody := []byte(`{"username":"alice","password":"password1"}`)
	loginRes := httptest.NewRecorder()
	server.Routes().ServeHTTP(loginRes, httptest.NewRequest(http.MethodPost, "/api/auth/login", bytes.NewReader(loginBody)))
	if loginRes.Code != http.StatusOK {
		t.Fatalf("login status = %d body=%s", loginRes.Code, loginRes.Body.String())
	}
}

func TestRegisterRejectsDuplicateUsernameAndNickname(t *testing.T) {
	server := testServer(t)
	first := []byte(`{"username":"alice","password":"password1","nickname":"Alice"}`)
	res1 := httptest.NewRecorder()
	server.Routes().ServeHTTP(res1, httptest.NewRequest(http.MethodPost, "/api/auth/register", bytes.NewReader(first)))
	if res1.Code != http.StatusCreated { t.Fatalf("first register status = %d", res1.Code) }

	dupUser := httptest.NewRecorder()
	server.Routes().ServeHTTP(dupUser, httptest.NewRequest(http.MethodPost, "/api/auth/register", bytes.NewReader([]byte(`{"username":"alice","password":"password2","nickname":"Alice2"}`))))
	if dupUser.Code != http.StatusConflict {
		t.Fatalf("duplicate username status = %d body=%s", dupUser.Code, dupUser.Body.String())
	}

	dupNick := httptest.NewRecorder()
	server.Routes().ServeHTTP(dupNick, httptest.NewRequest(http.MethodPost, "/api/auth/register", bytes.NewReader([]byte(`{"username":"bob","password":"password2","nickname":"Alice"}`))))
	if dupNick.Code != http.StatusConflict {
		t.Fatalf("duplicate nickname status = %d body=%s", dupNick.Code, dupNick.Body.String())
	}
}

func TestRegisterRejectsShortPasswordAndMeRequiresAuth(t *testing.T) {
	server := testServer(t)
	shortRes := httptest.NewRecorder()
	server.Routes().ServeHTTP(shortRes, httptest.NewRequest(http.MethodPost, "/api/auth/register", bytes.NewReader([]byte(`{"username":"alice","password":"short","nickname":"Alice"}`))))
	if shortRes.Code != http.StatusBadRequest {
		t.Fatalf("short password status = %d body=%s", shortRes.Code, shortRes.Body.String())
	}

	meRes := httptest.NewRecorder()
	server.Routes().ServeHTTP(meRes, httptest.NewRequest(http.MethodGet, "/api/me", nil))
	if meRes.Code != http.StatusUnauthorized {
		t.Fatalf("unauthorized me status = %d body=%s", meRes.Code, meRes.Body.String())
	}
}
