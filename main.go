package main

import (
	"context"
	"encoding/base64"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"time"

	"github.com/gorilla/websocket"

	dialogflowcx "cloud.google.com/go/dialogflow/cx/apiv3"
	"cloud.google.com/go/dialogflow/cx/apiv3/cxpb"
	"google.golang.org/api/option"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

var clients = make(map[*websocket.Conn]bool)
var broadcast = make(chan Message)

// Message represents a chat message
type Message struct {
	Username string `json:"username"`
	Message  string `json:"message"`
}

var botAgentMap = map[string]string{
	"/bot1": "9a9d4f03-3ca9-4517-b653-ff0843045cee", // Travel - Flight Information
	"/bot2": "df680c7d-6fc9-4e3c-a28f-bd2ca88e03ba", // Small Talk
	"/bot3": "4fb51b11-e84a-47bc-99f9-d36cbf2a913b", // Telecommunications
	"/bot4": "acd70926-641c-4984-917a-b8062243a38d", // Financial Services
	"/bot5": "cb9714ef-eac1-44ea-96d3-18befcfcaed8", // Payment Arrangement
	"/bot6": "14abc25d-229e-4119-b596-534dac48607b", // Order and Account Management
	"/bot7": "84bbfb0f-624e-4e72-802b-3469cfaefa9f", // Healthcare
	"/bot8": "d1aa5bec-e6ea-4778-917a-cd366c571bcc", // Baggage Claim
	"/bot9": "1e0d311c-73b5-4770-828d-83a6d3a4a9df", // Car Rental
}

func secureHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Set security-related headers
		w.Header().Set("Content-Security-Policy", "default-src 'self'")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Strict-Transport-Security", "max-age=63072000; includeSubDomains; preload")
		w.Header().Set("X-XSS-Protection", "1; mode=block")
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")

		// Call the next handler
		next.ServeHTTP(w, r)
	})
}

func main() {
	// Secure headers applied to all routes
	http.Handle("/", secureHeaders(http.HandlerFunc(serveHome)))
	http.Handle("/ws", secureHeaders(http.HandlerFunc(handleConnections)))
	http.Handle("/static/", secureHeaders(http.StripPrefix("/static/", http.FileServer(http.Dir("static")))))

	go handleMessages()

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080" // Default to 8080 if no port is specified
	}
	log.Println("Server starting on port:", port)
	err := http.ListenAndServe(":"+port, nil)
	if err != nil {
		log.Fatalf("Error starting server: %v", err)
	}
}

func serveHome(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "static/index.html")
}

var (
	messageRateLimit  = 100 * time.Millisecond // Minimum 100ms between messages
	clientLastMessage = make(map[*websocket.Conn]time.Time)
	maxMessageSize    = 1024            // Limit the size of incoming messages to 1KB
	messageCharLimit  = 500             // Limit the character length of a message
	connectionTimeout = 5 * time.Minute // Timeout for read operations
)

// Rate limiter for messages
func canSendMessage(conn *websocket.Conn) bool {
	lastMessageTime, exists := clientLastMessage[conn]
	if !exists || time.Since(lastMessageTime) > messageRateLimit {
		clientLastMessage[conn] = time.Now()
		return true
	}
	return false
}

// Sanitize user messages (example: trim spaces, remove unwanted characters)
func sanitizeMessage(input string) string {
	// Add specific sanitization logic as needed
	return input // Here we simply return the input; customize as necessary
}

var ipConnectionCount = make(map[string]int)
var maxConnectionsPerIP = 8 // Limit to 5 connections per IP

func handleConnections(w http.ResponseWriter, r *http.Request) {
	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		http.Error(w, "Invalid client address", http.StatusBadRequest)
		return
	}

	// Increment the count for this IP
	ipConnectionCount[ip]++
	defer func() {
		ipConnectionCount[ip]--
	}()

	if ipConnectionCount[ip] > maxConnectionsPerIP {
		http.Error(w, "Bad Request: Too many connections", http.StatusTooManyRequests)
		return
	}

	conn, err := upgrader.Upgrade(w, r, nil) // Upgrade the HTTP connection to WebSocket
	if err != nil {
		log.Printf("WebSocket upgrade error: %v", err)
		return
	}
	defer conn.Close() // Ensure the connection is closed when the function exits

	clients[conn] = true // Add the new client to the list of active connections
	log.Println("New client connected")
	defer delete(clients, conn) // Remove the client when they disconnect

	sessionID := fmt.Sprintf("session-%d", time.Now().UnixNano()) // Create a unique session ID

	// Configure WebSocket read limits and timeout
	conn.SetReadLimit(int64(maxMessageSize))                // Set max message size
	conn.SetReadDeadline(time.Now().Add(connectionTimeout)) // Set initial timeout for read operations
	conn.SetPongHandler(func(string) error {                // Reset the timeout on pong
		conn.SetReadDeadline(time.Now().Add(connectionTimeout))
		return nil
	})

	for {
		var msg Message
		// Read the message from the WebSocket
		err := conn.ReadJSON(&msg)
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("WebSocket read error: %v", err)
			}
			break
		}

		// Check if the user is sending messages too quickly
		if !canSendMessage(conn) {
			conn.WriteMessage(websocket.TextMessage, []byte("You are sending messages too quickly. Please slow down."))
			continue
		}

		// Check for excessive message length
		if len(msg.Message) > messageCharLimit {
			log.Printf("Message too long from user: %s", msg.Username)
			conn.WriteMessage(websocket.TextMessage, []byte("Message is too long. Limit to 500 characters."))
			continue
		}

		// Sanitize the message content
		msg.Message = sanitizeMessage(msg.Message)

		// Broadcast the user's message to all clients
		broadcast <- Message{Username: msg.Username, Message: msg.Message}

		// Check if the message is a bot command (e.g., "/bot1 Hello!")
		if len(msg.Message) >= 5 && msg.Message[:4] == "/bot" {
			botPrefix := msg.Message[:5] // Extract the bot command (e.g., "/bot1")
			agentID, exists := botAgentMap[botPrefix]
			if !exists {
				// If the bot command is invalid, notify the user
				broadcast <- Message{Username: "Bot", Message: "Invalid bot command. Use /bot1 to /bot9."}
				continue
			}

			// Remove the bot prefix and process the remaining message as a query
			userMessage := msg.Message[5:]
			botResponses, err := queryDialogflow(sessionID, userMessage, agentID)
			if err != nil {
				log.Printf("Dialogflow error: %v", err)
				broadcast <- Message{Username: "Bot", Message: "Sorry, I couldn't process your request."}
				continue
			}

			// Broadcast all bot responses to the chat
			for _, botResponse := range botResponses {
				broadcast <- Message{Username: "Bot", Message: botResponse}
			}
		}
	}
}

func handleMessages() {
	for msg := range broadcast {
		for client := range clients {
			err := client.WriteJSON(msg)
			if err != nil {
				log.Printf("WebSocket write error: %v", err)
				client.Close()
				delete(clients, client)
			}
		}
	}
}

func queryDialogflow(sessionID, message, agentID string) ([]string, error) {
	ctx := context.Background()

	// Configure credentials
	credentialsBase64 := os.Getenv("DIALOGFLOW_CREDENTIALS")
	if credentialsBase64 == "" {
		return nil, fmt.Errorf("DIALOGFLOW_CREDENTIALS environment variable is not set")
	}

	credentials, err := base64.StdEncoding.DecodeString(credentialsBase64)
	if err != nil {
		return nil, fmt.Errorf("failed to decode credentials: %v", err)
	}

	// Configure endpoint explicitly for us-central1
	clientOptions := []option.ClientOption{
		option.WithCredentialsJSON(credentials),
		option.WithEndpoint("us-central1-dialogflow.googleapis.com:443"),
	}

	// Create Dialogflow client
	client, err := dialogflowcx.NewSessionsClient(ctx, clientOptions...)
	if err != nil {
		return nil, fmt.Errorf("failed to create Dialogflow CX client: %v", err)
	}
	defer client.Close()

	// Define the session path dynamically based on the agent ID
	projectID := "go-chat-bot-435203"
	location := "us-central1"
	sessionPath := fmt.Sprintf("projects/%s/locations/%s/agents/%s/sessions/%s", projectID, location, agentID, sessionID)

	// Create a text input
	textInput := &cxpb.TextInput{
		Text: message,
	}
	queryInput := &cxpb.QueryInput{
		Input: &cxpb.QueryInput_Text{
			Text: textInput,
		},
		LanguageCode: "en",
	}

	// Send the query to Dialogflow CX
	response, err := client.DetectIntent(ctx, &cxpb.DetectIntentRequest{
		Session:    sessionPath,
		QueryInput: queryInput,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to detect intent: %v", err)
	}

	// Extract all response messages
	var responses []string
	for _, message := range response.GetQueryResult().GetResponseMessages() {
		if text := message.GetText().GetText(); len(text) > 0 {
			responses = append(responses, text[0]) // Append each message to the list
		}
	}

	if len(responses) == 0 {
		return nil, fmt.Errorf("no response from Dialogflow CX")
	}

	return responses, nil
}
