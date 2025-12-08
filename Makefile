build:
	make fmt
	make libs
	CGO_ENABLED=0 go build -o ./cmd/agent ./cmd/agent
	CGO_ENABLED=0 go build -o ./cmd/server ./cmd/server
	CGO_ENABLED=0 go build -o ./cmd/staticlint ./cmd/staticlint
	CGO_ENABLED=0 go build -o ./cmd/reset ./cmd/reset
libs:
	go mod tidy
	go mod vendor
test:
	make libs
	make build
	CGO_ENABLED=0 sh test.sh
load_test:
	sh load_test.sh http://localhost:8080 30000
bench:
	go test -bench=. -benchmem ./...
start_server:
	make libs
	make build
	./cmd/server/server 
get_profile:
	curl http://localhost:8080/debug/pprof/heap > profiles/profile_after.pprof
get_profiles_diff:
	go tool pprof -top -diff_base=profiles/profile_before.pprof  profiles/profile_after.pprof
fmt:
	go fmt ./...
coverage:
	sh coverage.sh
staticlint:
	./cmd/staticlint/staticlint ./...