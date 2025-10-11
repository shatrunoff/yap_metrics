build:
	CGO_ENABLED=0 go build -o ./cmd/agent ./cmd/agent
	CGO_ENABLED=0 go build -o ./cmd/server ./cmd/server
libs:
	go mod tidy
	go mod vendor
test:
	make libs
	make build
	CGO_ENABLED=0 sh test.sh