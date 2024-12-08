# Go-Chat Project

A chatroom project made for learning about Web Sockets, Containerization, and Cloud Deployment.

## Instructions:

### Clone the Repository:
Run the following command: `git clone https://github.com/collaboratorsje/Go-Chat.git`

### Navigate to the Directory:
Run the following command: `cd go-chat`

### Run the Server:

Ensure you have Golang installed on your system.

Run the following command: `go run main.go`

The server will be live on `localhost:8080`.

This chat is still under development.

## Dependencies:

### Core Dependencies:

Install the following packages to ensure the application runs correctly:
- `go get github.com/gorilla/websocket`
- `go get cloud.google.com/go/dialogflow/cx/apiv3`
- `go get google.golang.org/api/option`
- `go get github.com/joho/godotenv`

### Testing Dependencies:

Install the following packages and tools to run tests and view coverage reports:
- `go get github.com/stretchr/testify`
- `go install github.com/golang/mock/mockgen@latest`
- `go install golang.org/x/tools/cmd/cover@latest`

## Live Server:

You can access the live version of the application here: [Live Server Link](https://go-chat-1036147648426.us-central1.run.app/)