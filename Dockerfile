FROM golang:1.23-alpine

    WORKDIR /app

    COPY . .
    RUN go mod download
    RUN go mod tidy

    #
    RUN go install github.com/githubnemo/CompileDaemon@latest
    CMD CompileDaemon -polling -polling-interval=1000 -build="go build -o bin/main main.go" -command="./bin/main"