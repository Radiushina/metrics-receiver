BINARY = metrics-server

.PHONY: run lint run-server run-agent run-all build build-server build-agent migrate-postgres test-iter1 test-iter2 test-iter3 test-iter7 test-iter8 test-iter9 test-iter11 test-iter12 unit

lint:
	golangci-lint run --config .golangci.yml

run: run-server

run-server:
	go run ./cmd/server/

run-agent:
	go run ./cmd/agent/

run-all:
	@echo "Starting server in background, then agent (Ctrl+C stops both)"
	@bash -c 'set -e; go run ./cmd/server/ & srv=$$!; trap "kill $$srv 2>/dev/null; wait $$srv 2>/dev/null" EXIT INT TERM; sleep 1; go run ./cmd/agent/'

build-server:
	go build -buildvcs=false -o cmd/server/server ./cmd/server

build-agent:
	go build -buildvcs=false -o cmd/agent/agent ./cmd/agent

migrate-postgres:
ifneq "$(name)" ""
	migrate create -ext sql -dir migrations $(name)
else
	echo "\nSpecify migration script name\n";
endif

test-iter1: build-server
	./metricstest -test.v -test.run=^TestIteration1$$ -binary-path=cmd/server/server

# As in .github/workflows/mertricstest.yml (increment #2): regex picks TestIteration2, TestIteration2A, TestIteration2B, …
test-iter2: build-agent
	./metricstest -test.v -test.run=^TestIteration2[AB]*$$ -source-path=. -agent-binary-path=cmd/agent/agent

test-iter3: build-server build-agent
	./metricstest -test.v -test.run=^TestIteration3[AB]*$$ -source-path=. -agent-binary-path=cmd/agent/agent -binary-path=cmd/server/server

# Same flags as .github/workflows/mertricstest.yml ("Code increment #7").
# SERVER_PORT=8080, ADDRESS=localhost:8080 — override port: ITER7_SERVER_PORT=9090 make test-iter7
test-iter7: build-server build-agent
	@SERVER_PORT="$${ITER7_SERVER_PORT:-8080}"; \
	export SERVER_PORT; \
	export ADDRESS="localhost:$$SERVER_PORT"; \
	export TEMP_FILE="$$(mktemp)"; \
	./metricstest -test.v -test.run='^TestIteration7$$' \
		-agent-binary-path=cmd/agent/agent \
		-binary-path=cmd/server/server \
		-server-port="$$SERVER_PORT" \
		-source-path=.

test-iter8: build-server build-agent
	@SERVER_PORT="$${ITER8_SERVER_PORT:-8080}"; \
	export SERVER_PORT; \
	export ADDRESS="localhost:$$SERVER_PORT"; \
	export TEMP_FILE="$$(mktemp)"; \
	./metricstest -test.v -test.run='^TestIteration8$$' \
		-agent-binary-path=cmd/agent/agent \
		-binary-path=cmd/server/server \
		-server-port="$$SERVER_PORT" \
		-source-path=.

test-iter9: build-server build-agent
	@SERVER_PORT="$${ITER9_SERVER_PORT:-8080}"; \
	export SERVER_PORT; \
	export ADDRESS="localhost:$$SERVER_PORT"; \
	export TEMP_FILE="$$(mktemp)"; \
	./metricstest -test.v -test.run='^TestIteration9$$' \
		-agent-binary-path=cmd/agent/agent \
		-binary-path=cmd/server/server \
		-file-storage-path="$$TEMP_FILE" \
		-server-port="$$SERVER_PORT" \
		-source-path=.

# Нужен доступный PostgreSQL.
# Порт сервера: ITER11_SERVER_PORT (по умолчанию 8080). DSN: ITER11_DATABASE_DSN.
test-iter11: build-server build-agent
	@SERVER_PORT="$${ITER11_SERVER_PORT:-8080}"; \
	export SERVER_PORT; \
	export ADDRESS="localhost:$$SERVER_PORT"; \
	DSN="$${ITER11_DATABASE_DSN:-postgres://developer@localhost:5432/metrics?sslmode=disable}"; \
	./metricstest -test.v -test.run='^TestIteration11$$' \
		-agent-binary-path=cmd/agent/agent \
		-binary-path=cmd/server/server \
		-database-dsn="$$DSN" \
		-server-port="$$SERVER_PORT" \
		-source-path=.

# Нужен доступный PostgreSQL.
# Порт: ITER12_SERVER_PORT (по умолчанию 8080). DSN: ITER12_DATABASE_DSN.
test-iter12: build-server build-agent
	@cd "$(CURDIR)" && \
	SERVER_PORT="$${ITER12_SERVER_PORT:-8086}" && \
	export SERVER_PORT ADDRESS="localhost:$$SERVER_PORT" && \
	DSN="$${ITER12_DATABASE_DSN:-postgres://developer@localhost:5432/metrics?sslmode=disable}" && \
	./metricstest -test.v -test.run='^TestIteration12$$' \
		-agent-binary-path="$(CURDIR)/cmd/agent/agent" \
		-binary-path="$(CURDIR)/cmd/server/server" \
		-database-dsn="$$DSN" \
		-server-port="$$SERVER_PORT" \
		-source-path="$(CURDIR)"


unit:
	go test ./...
