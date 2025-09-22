build:
	go build -o ./cmd/agent ./cmd/agent
	go build -o ./cmd/server ./cmd/server
test:
	make build
	sh test.sh
