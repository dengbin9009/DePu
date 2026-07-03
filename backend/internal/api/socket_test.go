package api

import (
	"bufio"
	"bytes"
	"encoding/json"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

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

	conn, reader := dialSocket(t, ts.URL, "/api/socket?token="+token)
	defer conn.Close()

	readUpgradeResponse(t, reader)
	_, payload := readServerTextFrame(t, reader)
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

	conn, reader := dialSocket(t, ts.URL, "/api/socket?token="+token)
	defer conn.Close()
	readUpgradeResponse(t, reader)
	readServerTextFrame(t, reader)

	if err := writeClientTextFrame(conn, []byte(`{"type":"room.refresh","requestId":"req_keepalive"}`)); err != nil {
		t.Fatal(err)
	}
	if err := writeClientCloseFrame(conn); err != nil {
		t.Fatal(err)
	}
}

func TestSocketHubTracksActiveConnectionUntilClientCloses(t *testing.T) {
	server := testServer(t)
	token := registerUser(t, server, "socket_tracked", "Socket追踪")
	ts := httptest.NewServer(server.Routes())
	defer ts.Close()

	conn, reader := dialSocket(t, ts.URL, "/api/socket?token="+token)
	defer conn.Close()
	readUpgradeResponse(t, reader)
	readServerTextFrame(t, reader)
	time.Sleep(20 * time.Millisecond)

	if got := socketHubClientCount(server.hub); got != 1 {
		t.Fatalf("active socket clients = %d, want 1", got)
	}
	if err := writeClientCloseFrame(conn); err != nil {
		t.Fatal(err)
	}
	eventually(t, func() bool { return socketHubClientCount(server.hub) == 0 })
}

func socketUpgradeRequest(path string) *http.Request {
	req := httptest.NewRequest(http.MethodGet, path, nil)
	req.Header.Set("Connection", "Upgrade")
	req.Header.Set("Upgrade", "websocket")
	req.Header.Set("Sec-WebSocket-Version", "13")
	req.Header.Set("Sec-WebSocket-Key", "dGhlIHNhbXBsZSBub25jZQ==")
	return req
}

func dialSocket(t *testing.T, serverURL, path string) (net.Conn, *bufio.Reader) {
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
	return conn, bufio.NewReader(conn)
}

func readUpgradeResponse(t *testing.T, reader *bufio.Reader) {
	t.Helper()
	statusLine, err := reader.ReadString('\n')
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(statusLine, "101 Switching Protocols") {
		t.Fatalf("status line = %q, want websocket upgrade", statusLine)
	}
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			t.Fatal(err)
		}
		if line == "\r\n" {
			return
		}
	}
}

func readServerTextFrame(t *testing.T, reader *bufio.Reader) (byte, []byte) {
	t.Helper()
	header := make([]byte, 2)
	if _, err := reader.Read(header); err != nil {
		t.Fatal(err)
	}
	opcode := header[0] & 0x0f
	length := int(header[1] & 0x7f)
	if length == 126 {
		extended := make([]byte, 2)
		if _, err := reader.Read(extended); err != nil {
			t.Fatal(err)
		}
		length = int(extended[0])<<8 | int(extended[1])
	}
	payload := make([]byte, length)
	if _, err := reader.Read(payload); err != nil {
		t.Fatal(err)
	}
	if opcode != 1 {
		t.Fatalf("opcode = %d, want text frame", opcode)
	}
	return opcode, payload
}

func writeClientTextFrame(conn net.Conn, payload []byte) error {
	return writeClientFrame(conn, 0x1, payload)
}

func writeClientCloseFrame(conn net.Conn) error {
	return writeClientFrame(conn, 0x8, nil)
}

func writeClientFrame(conn net.Conn, opcode byte, payload []byte) error {
	mask := []byte{1, 2, 3, 4}
	header := []byte{0x80 | opcode, 0x80 | byte(len(payload))}
	masked := make([]byte, len(payload))
	for i, b := range payload {
		masked[i] = b ^ mask[i%len(mask)]
	}
	frame := append(header, mask...)
	frame = append(frame, masked...)
	_, err := conn.Write(frame)
	return err
}

func socketHubClientCount(hub *socketHub) int {
	hub.mu.RLock()
	defer hub.mu.RUnlock()
	return len(hub.clients)
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
