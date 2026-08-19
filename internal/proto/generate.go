// Package proto содержит gRPC-контракт сервиса метрик и сгенерированный код.
package proto

//go:generate protoc --go_out=. --go_opt=paths=source_relative --go-grpc_out=. --go-grpc_opt=paths=source_relative metrics.proto
