document.addEventListener('DOMContentLoaded', function() {
    const chat = document.getElementById('chat');
    const messageInput = document.getElementById('messageInput');
    const usernameModal = document.getElementById('usernameModal');
    const usernameInput = document.getElementById('usernameInput');
    const joinChatButton = document.getElementById('joinChatButton');
    let username;

    const ws = new WebSocket('ws://localhost:8080/ws');

    ws.onmessage = function(event) {
        const message = JSON.parse(event.data);
        const messageElement = document.createElement('div');
        messageElement.innerHTML = `<strong>${message.username}:</strong> ${message.message.replace(/\n/g, '<br>')}`;
        chat.appendChild(messageElement);
        chat.scrollTop = chat.scrollHeight; // Auto-scroll to the bottom
    };

    function sendMessage() {
        if (!username) {
            alert("Please enter your username");
            return;
        }
        const message = {
            username: username,
            message: messageInput.value
        };
        ws.send(JSON.stringify(message));
        messageInput.value = '';
    }

    function setUsername() {
        username = usernameInput.value.trim();
        if (username) {
            usernameModal.style.display = 'none';
        } else {
            alert("Username cannot be empty");
        }
    }

    joinChatButton.addEventListener('click', setUsername);

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
