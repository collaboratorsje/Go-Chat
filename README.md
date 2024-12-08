A chatroom project made for learning about Web Sockets, Containerization, and Cloud Deployment.

Instructions:

Clone repo
Navigate to directory you cloned it into
Command (with Golang installed): go run main.go
server will be live on localhost:8080

This chat is still under development.

go get github.com/gorilla/websocket
go get cloud.google.com/go/dialogflow/cx/apiv3
go get google.golang.org/api/option

Testing

go get github.com/stretchr/testify
go install github.com/golang/mock/mockgen@latest
go install golang.org/x/tools/cmd/cover@latest

Live Server:
https://go-chat-1036147648426.us-central1.run.app/