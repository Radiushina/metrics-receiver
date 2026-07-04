FROM golang:1.26-alpine AS builder

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -buildvcs=false -o /bin/server ./cmd/server
RUN CGO_ENABLED=0 GOOS=linux go build -buildvcs=false -o /bin/agent ./cmd/agent

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
