let username;
const ws = new WebSocket('ws://localhost:8080/ws');

document.addEventListener('DOMContentLoaded', function() {
    const chat = document.getElementById('chat');
    const messageInput = document.getElementById('messageInput');
    const usernameModal = document.getElementById('usernameModal');
    const usernameInput = document.getElementById('usernameInput');
    const joinChatButton = document.getElementById('joinChatButton');

    ws.onopen = function() {
        console.log("WebSocket connection established");
    };

    ws.onmessage = function(event) {
        const message = JSON.parse(event.data);
        const messageElement = document.createElement('div');
        messageElement.innerHTML = `<strong>${message.username}:</strong> ${message.message.replace(/\n/g, '<br>')}`;
        chat.appendChild(messageElement);
        chat.scrollTop = chat.scrollHeight; // Auto-scroll to the bottom
        console.log("Message received:", message);
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

    // Handle keypress events for the textarea
    messageInput.addEventListener('keydown', function(event) {
        if (event.key === 'Enter' && !event.shiftKey) {
            event.preventDefault(); // Prevent new line
            sendMessage(); // Send the message
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
    } else {
        alert("Username cannot be empty");
    }
}
