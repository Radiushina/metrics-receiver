BINARY = metrics-server

.PHONY: run dev build build-server test

run:
	go run ./cmd/server/main.go

build-server:
	go build -o cmd/server/server cmd/server/*.go

test:
	./metricstest -test.v -test.run=^TestIteration1$$ -binary-path=cmd/server/server