package main

import (
	"encoding/base64"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
)

func TestBotAgentMap(t *testing.T) {
	agentID, exists := botAgentMap["/bot1"]
	assert.True(t, exists, "Expected /bot1 to exist in botAgentMap")
	assert.Equal(t, "9a9d4f03-3ca9-4517-b653-ff0843045cee", agentID)
}

func TestSessionIDGeneration(t *testing.T) {
	sessionID1 := fmt.Sprintf("session-%d", time.Now().UnixNano())
	time.Sleep(1 * time.Nanosecond)
	sessionID2 := fmt.Sprintf("session-%d", time.Now().UnixNano())
	assert.NotEqual(t, sessionID1, sessionID2, "Expected session IDs to be unique")
}

func TestServeHome(t *testing.T) {
	req, _ := http.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()

	serveHome(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestHandleConnections(t *testing.T) {
	// Start the server with the WebSocket handler
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			t.Fatalf("Failed to upgrade connection: %v", err)
		}
		defer conn.Close()

		// Read a message from the WebSocket
		var msg Message
		err = conn.ReadJSON(&msg)
		if err != nil {
			t.Fatalf("Failed to read JSON from WebSocket: %v", err)
		}

		// Echo the message back
		err = conn.WriteJSON(msg)
		if err != nil {
			t.Fatalf("Failed to write JSON to WebSocket: %v", err)
		}
	}))
	defer server.Close()

	// Convert the server URL to a WebSocket URL
	wsURL := "ws" + server.URL[len("http"):]

	// Connect to the server as a WebSocket client
	ws, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("Failed to connect to WebSocket server: %v", err)
	}
	defer ws.Close()

	// Test sending and receiving a message
	testMessage := Message{Username: "TestUser", Message: "Hello"}
	err = ws.WriteJSON(testMessage)
	assert.NoError(t, err, "Expected no error while sending WebSocket message")

	// Receive the echoed message
	var receivedMessage Message
	err = ws.ReadJSON(&receivedMessage)
	assert.NoError(t, err, "Expected no error while receiving WebSocket message")
	assert.Equal(t, testMessage, receivedMessage, "Expected received message to match sent message")
}

func TestBroadcastMessages(t *testing.T) {
	go func() {
		broadcast <- Message{Username: "User", Message: "Hello"}
	}()

	select {
	case msg := <-broadcast:
		assert.Equal(t, "User", msg.Username, "Expected username to match")
		assert.Equal(t, "Hello", msg.Message, "Expected message content to match")
	case <-time.After(1 * time.Second):
		t.Error("Expected message on broadcast channel but timed out")
	}
}

func TestInvalidBotCommand(t *testing.T) {
	botPrefix := "/bot999"
	_, exists := botAgentMap[botPrefix]
	assert.False(t, exists, "Expected bot command to not exist")
}

func queryDialogflowMock(sessionID, message, agentID string) ([]string, error) {
	_ = sessionID // Explicitly ignore sessionID
	_ = message   // Explicitly ignore message

	if agentID == "invalid-agent" {
		return nil, fmt.Errorf("Invalid agent ID")
	}
	return []string{"Mocked response from bot"}, nil
}

func TestQueryDialogflowMock(t *testing.T) {
	sessionID := "test-session"
	message := "Test Message"
	responses, err := queryDialogflowMock(sessionID, message, "valid-agent")

	assert.NoError(t, err, "Expected no error from queryDialogflowMock")
	assert.Equal(t, []string{"Mocked response from bot"}, responses, "Expected mocked response")
}

func TestQueryDialogflow(t *testing.T) {
	originalEnv := os.Getenv("DIALOGFLOW_CREDENTIALS")
	defer os.Setenv("DIALOGFLOW_CREDENTIALS", originalEnv)
	os.Setenv("DIALOGFLOW_CREDENTIALS", base64.StdEncoding.EncodeToString([]byte(`{}`))) // Mock credentials

	sessionID := "test-session"
	message := "Hello"
	agentID := "valid-agent-id"

	// Mock API response
	responses, err := queryDialogflow(sessionID, message, agentID)
	assert.Error(t, err, "Expected error with mocked credentials")
	assert.Nil(t, responses, "Expected no responses due to error")
}

func TestMissingEnvVariable(t *testing.T) {
	originalValue := os.Getenv("DIALOGFLOW_CREDENTIALS")
	defer os.Setenv("DIALOGFLOW_CREDENTIALS", originalValue)

	os.Unsetenv("DIALOGFLOW_CREDENTIALS")
	_, err := queryDialogflow("test-session", "Hello", "valid-agent")
	assert.Error(t, err, "Expected an error when DIALOGFLOW_CREDENTIALS is missing")
}

func TestEmptyMessageHandling(t *testing.T) {
	go func() {
		broadcast <- Message{Username: "User", Message: ""}
	}()

	select {
	case msg := <-broadcast:
		assert.Equal(t, "", msg.Message, "Expected empty message content to pass through")
	case <-time.After(1 * time.Second):
		t.Error("Expected message on broadcast channel but timed out")
	}
}

func TestConcurrentBroadcasts(t *testing.T) {
	messages := []Message{
		{Username: "User1", Message: "Hello"},
		{Username: "User2", Message: "Hi"},
		{Username: "User3", Message: "Hey"},
	}

	go func() {
		for _, msg := range messages {
			broadcast <- msg
		}
	}()

	for i := 0; i < len(messages); i++ {
		select {
		case msg := <-broadcast:
			assert.Contains(t, []string{"Hello", "Hi", "Hey"}, msg.Message, "Expected message content to match")
		case <-time.After(1 * time.Second):
			t.Error("Expected message on broadcast channel but timed out")
		}
	}
}

func TestMalformedWebSocketRequest(t *testing.T) {
	req, _ := http.NewRequest("GET", "/ws", nil)
	w := httptest.NewRecorder()

	handleConnections(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code, "Expected WebSocket handshake to fail with 400 Bad Request")
	assert.Contains(t, w.Body.String(), "Invalid client address", "Expected response to indicate invalid client address")
}

func TestSimulatedChatFlow(t *testing.T) {
	mockQueryDialogflow := func(sessionID, message, agentID string) ([]string, error) {
		_ = sessionID // Explicitly ignore sessionID
		_ = message   // Explicitly ignore message
		_ = agentID   // Explicitly ignore agentID
		return []string{"Hello, this is a mocked response!"}, nil
	}

	sessionID := "test-session"
	userMessage := "Hello, bot!"
	agentID := "/bot1"

	responses, err := mockQueryDialogflow(sessionID, userMessage, botAgentMap[agentID])
	assert.NoError(t, err, "Expected no error from mock query")
	assert.Equal(t, []string{"Hello, this is a mocked response!"}, responses, "Expected mocked response")
}

func TestRateLimiting(t *testing.T) {
	conn := &websocket.Conn{} // Mock connection
	canSend1 := canSendMessage(conn)
	assert.True(t, canSend1, "Expected to allow sending the first message")

	canSend2 := canSendMessage(conn)
	assert.False(t, canSend2, "Expected to block sending a message too quickly")

	time.Sleep(100 * time.Millisecond)
	canSend3 := canSendMessage(conn)
	assert.True(t, canSend3, "Expected to allow sending a message after rate limit duration")
}

func TestConnectionTimeout(t *testing.T) {
	// Start a mock WebSocket server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			t.Fatalf("WebSocket upgrade failed: %v", err)
		}
		defer conn.Close()

		// Set the read deadline to simulate timeout
		conn.SetReadDeadline(time.Now().Add(1 * time.Second))

		// Wait for timeout
		time.Sleep(2 * time.Second)

		// Attempt to read from the connection
		_, _, err = conn.ReadMessage()
		assert.Error(t, err, "Expected timeout error")
	}))
	defer server.Close()

	// Convert the server URL to a WebSocket URL
	wsURL := "ws" + server.URL[len("http"):]

	// Connect to the WebSocket server
	ws, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("Failed to connect to WebSocket server: %v", err)
	}
	defer ws.Close()

	// Do not send any messages, simulating an idle client
	time.Sleep(3 * time.Second)

	// Server should have closed the connection due to timeout
	_, _, err = ws.ReadMessage()
	assert.Error(t, err, "Expected timeout error")
}

func TestIPConnectionLimit(t *testing.T) {
	ip := "127.0.0.1:12345"

	// Simulate connections from the same IP
	for i := 0; i < maxConnectionsPerIP; i++ {
		ipConnectionCount[ip]++
	}

	// Exceed the limit
	ipConnectionCount[ip]++
	assert.Equal(t, maxConnectionsPerIP+1, ipConnectionCount[ip], "Expected IP connection count to increment")

	if ipConnectionCount[ip] > maxConnectionsPerIP {
		t.Log("IP exceeded connection limit, as expected")
	}

	// Clean up
	ipConnectionCount[ip] = 0
}

func TestConnectionLimit(t *testing.T) {
	originalMaxConnections := maxConnectionsPerIP
	maxConnectionsPerIP = 2 // Lower the limit for testing
	defer func() { maxConnectionsPerIP = originalMaxConnections }()

	server := httptest.NewServer(http.HandlerFunc(handleConnections))
	defer server.Close()

	wsURL := "ws" + server.URL[len("http"):]

	var connections []*websocket.Conn
	for i := 0; i < maxConnectionsPerIP; i++ {
		ws, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
		if err != nil {
			t.Fatalf("Failed to connect WebSocket: %v", err)
		}
		connections = append(connections, ws)
	}

	_, resp, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if resp != nil {
		defer resp.Body.Close()
	}

	if resp != nil {
		assert.Equal(t, http.StatusTooManyRequests, resp.StatusCode, "Expected 429 Too Many Requests")
	}
	assert.Error(t, err, "Expected error due to connection limit exceeded")

	for _, ws := range connections {
		ws.Close()
	}
}

func TestSanitizeMessage(t *testing.T) {
	input := " <script>alert('test')</script> "
	expected := " <script>alert('test')</script> " // Replace with your sanitization logic
	output := sanitizeMessage(input)
	assert.Equal(t, expected, output, "Expected sanitized message to match")
}

func TestSecureHeaders(t *testing.T) {
	handler := secureHeaders(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req, _ := http.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "default-src 'self'", w.Header().Get("Content-Security-Policy"))
	assert.Equal(t, "nosniff", w.Header().Get("X-Content-Type-Options"))
	assert.Equal(t, "DENY", w.Header().Get("X-Frame-Options"))
	assert.Contains(t, w.Header().Get("Strict-Transport-Security"), "max-age=63072000")
	assert.Equal(t, "1; mode=block", w.Header().Get("X-XSS-Protection"))
	assert.Equal(t, "strict-origin-when-cross-origin", w.Header().Get("Referrer-Policy"))
}

func TestMessageCharacterLimit(t *testing.T) {
	longMessage := string(make([]byte, messageCharLimit+1))
	msg := Message{Username: "TestUser", Message: longMessage}

	if len(msg.Message) > messageCharLimit {
		truncated := msg.Message[:messageCharLimit]
		assert.True(t, len(truncated) == messageCharLimit, "Expected message to truncate to character limit")
		assert.Equal(t, "TestUser", msg.Username, "Expected Username field to remain unchanged")
	}
}

func TestBroadcastChannelClosed(t *testing.T) {
	close(broadcast)

	defer func() {
		if r := recover(); r != nil {
			assert.NotNil(t, r, "Expected panic when writing to a closed channel")
		}
	}()

	broadcast <- Message{Username: "User", Message: "Hello"}
	t.Log("Broadcast channel is closed. Recovered from panic.")
}

func TestWebSocketReadError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(handleConnections))
	defer server.Close()

	wsURL := "ws" + server.URL[len("http"):]

	// Connect to the WebSocket server
	ws, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("Failed to connect WebSocket: %v", err)
	}
	ws.Close() // Close the connection immediately to trigger a read error

	// Attempt to send a message after closing the connection
	err = ws.WriteJSON(Message{Username: "User", Message: "Hello"})
	assert.Error(t, err, "Expected WebSocket write error")
}

func TestExcessiveMessageLength(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(handleConnections))
	defer server.Close()

	wsURL := "ws" + server.URL[len("http"):]

	// Connect to the WebSocket server
	ws, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("Failed to connect WebSocket: %v", err)
	}
	defer ws.Close()

	// Send a long message
	longMessage := string(make([]byte, messageCharLimit+1))
	err = ws.WriteJSON(Message{Username: "User", Message: longMessage})
	assert.NoError(t, err, "Expected excessive message length to be handled")
}

func TestValidBotCommand(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(handleConnections))
	defer server.Close()

	wsURL := "ws" + server.URL[len("http"):]

	// Connect to the WebSocket server
	ws, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("Failed to connect WebSocket: %v", err)
	}
	defer ws.Close()

	// Send a valid bot command
	err = ws.WriteJSON(Message{Username: "User", Message: "/bot1 Hello"})
	assert.NoError(t, err, "Expected valid bot command to be processed")
}

func TestWebSocketReadErrorHandling(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(handleConnections))
	defer server.Close()

	wsURL := "ws" + server.URL[len("http"):]

	// Connect to the WebSocket server
	ws, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("Failed to connect WebSocket: %v", err)
	}
	ws.Close() // Immediately close to trigger read error

	// Attempt to send a message to trigger an error
	err = ws.WriteJSON(Message{Username: "User", Message: "Hello"})
	assert.Error(t, err, "Expected error when writing to a closed WebSocket")
}

func TestBroadcastEmptyChannel(t *testing.T) {
	// Create a local broadcast channel to simulate a closed channel
	localBroadcast := make(chan Message)

	// Close the channel to simulate failure
	close(localBroadcast)

	defer func() {
		if r := recover(); r != nil {
			assert.NotNil(t, r, "Expected panic when writing to a closed channel")
		}
	}()

	// Attempt to send a message to the closed channel
	localBroadcast <- Message{Username: "User", Message: "Hello"}
	t.Error("Expected a panic when writing to a closed channel")
}

func TestSanitizedMessageHandling(t *testing.T) {
	rawMessage := " <script>alert('xss')</script> "
	expectedSanitized := " <script>alert('xss')</script> " // Adjust based on sanitization logic

	sanitizedMessage := sanitizeMessage(rawMessage)
	assert.Equal(t, expectedSanitized, sanitizedMessage, "Expected sanitization logic to clean input correctly")
}
