build:
	go build -o ./cmd/agent ./cmd/agent
	go build -o ./cmd/server ./cmd/server
libs:
	go mod tidy
	go mod vendor
test:
	make libs
	make build
	sh test.sh