let username;
const protocol = window.location.protocol === 'https:' ? 'wss' : 'ws';
const host = window.location.hostname;
const wsUrl = host === 'localhost'
    ? `${protocol}://${host}:8080/ws` // Use port 8080 for local development
    : `${protocol}://${host}/ws`;     // No port for Cloud Run

const ws = new WebSocket(wsUrl);

document.addEventListener('DOMContentLoaded', function() {
    const chat = document.getElementById('chat');
    const messageInput = document.getElementById('messageInput');
    const usernameModal = document.getElementById('usernameModal');
    const usernameInput = document.getElementById('usernameInput');
    const joinChatButton = document.getElementById('joinChatButton');
    const fontSelect = document.getElementById('fontSelect');
    let selectedFont = fontSelect.value;

    ws.onopen = function() {
        console.log("WebSocket connection established");
    };

    ws.onmessage = function(event) {
        const message = JSON.parse(event.data);
        const messageElement = document.createElement('div');
    
        // Add class based on message sender
        if (message.username === 'Bot') {
            messageElement.classList.add('bot-message');
        } else {
            messageElement.classList.add('user-message');
        }
    
        // Set message content
        messageElement.innerHTML = `<strong>${message.username}:</strong> ${message.message.replace(/\n/g, '<br>')}`;
        document.getElementById('chat').appendChild(messageElement);
    
        // Scroll to the latest message
        const chat = document.getElementById('chat');
        chat.scrollTop = chat.scrollHeight;
    };

    ws.onerror = function(error) {
        console.error("WebSocket error:", error);
    };

    ws.onclose = function() {
        console.log("WebSocket connection closed");
    };

    joinChatButton.addEventListener('click', function() {
        setUsername(usernameInput.value.trim());
    });

    fontSelect.addEventListener('change', function() {
        selectedFont = fontSelect.value;
        console.log("Selected font:", selectedFont);
    });

    // Handle keypress events for the textarea
    messageInput.addEventListener('keydown', function(event) {
        if (event.key === 'Enter' && !event.shiftKey) {
            event.preventDefault();
            sendMessage();
        }
    });

    // Show modal on load
    usernameModal.style.display = 'flex';
});

function sendMessage() {
    const messageInput = document.getElementById('messageInput');
    if (!username) {
        alert("Please enter your username");
        return;
    }
    const message = {
        username: username,
        message: messageInput.value
    };
    console.log("Sending message:", message);
    ws.send(JSON.stringify(message));
    messageInput.value = '';
}

function setUsername(name) {
    if (name) {
        username = name;
        const usernameModal = document.getElementById('usernameModal');
        usernameModal.style.display = 'none';
        console.log("Username set to:", username);
        ws.send(JSON.stringify({ username: username, message: "joined" }));
    } else {
        alert("Username cannot be empty");
    }
}

function resetSession() {
    if (ws.readyState === WebSocket.OPEN) {
        const resetMessage = {
            username: "System",
            message: "/reset"
        };
        ws.send(JSON.stringify(resetMessage));
        console.log("Session reset command sent");
    } else {
        console.error("WebSocket is not open");
    }
}




