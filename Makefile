BINARY = metrics-server

.PHONY: run run-server run-agent run-all build build-server build-agent test-iter1 test-iter2 unit

run: run-server

run-server:
	go run ./cmd/server/main.go

run-agent:
	go run ./cmd/agent/main.go

run-all:
	@echo "Starting server in background, then agent (Ctrl+C stops both)"
	@bash -c 'set -e; go run ./cmd/server/main.go & srv=$$!; trap "kill $$srv 2>/dev/null; wait $$srv 2>/dev/null" EXIT INT TERM; sleep 1; go run ./cmd/agent/main.go'

build-server:
	go build -buildvcs=false -o cmd/server/server ./cmd/server

build-agent:
	go build -buildvcs=false -o cmd/agent/agent ./cmd/agent

test-iter1: build-server
	./metricstest -test.v -test.run=^TestIteration1$$ -binary-path=cmd/server/server

# As in .github/workflows/mertricstest.yml (increment #2): regex picks TestIteration2, TestIteration2A, TestIteration2B, …
test-iter2: build-agent
	./metricstest -test.v -test.run=^TestIteration2[AB]*$$ -source-path=. -agent-binary-path=cmd/agent/agent

unit:
	go test ./...