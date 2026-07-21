FROM golang:1.26-alpine AS builder

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download

COPY . .

# Метаданные сборки: docker build --build-arg VERSION=1.2.3 --build-arg COMMIT=abc1234 .
ARG VERSION=N/A
ARG DATE=N/A
ARG COMMIT=N/A
RUN CGO_ENABLED=0 GOOS=linux go build -buildvcs=false \
	-ldflags="-X main.buildVersion=${VERSION} -X main.buildDate=${DATE} -X main.buildCommit=${COMMIT}" \
	-o /bin/server ./cmd/server
RUN CGO_ENABLED=0 GOOS=linux go build -buildvcs=false \
	-ldflags="-X main.buildVersion=${VERSION} -X main.buildDate=${DATE} -X main.buildCommit=${COMMIT}" \
	-o /bin/agent ./cmd/agent

FROM alpine:3.20 AS server

RUN apk add --no-cache ca-certificates wget

WORKDIR /app
COPY --from=builder /bin/server ./server
COPY migrations ./migrations

EXPOSE 8080
ENTRYPOINT ["./server"]

FROM alpine:3.20 AS agent

RUN apk add --no-cache ca-certificates

WORKDIR /app
COPY --from=builder /bin/agent ./agent

ENTRYPOINT ["./agent"]
