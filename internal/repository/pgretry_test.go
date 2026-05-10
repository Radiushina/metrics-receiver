package repository

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5/pgconn"
)

func TestRetryPostgres_RetriesConnectionFailure(t *testing.T) {
	old := pgRetrySleep
	pgRetrySleep = func(context.Context, time.Duration) error { return nil }
	t.Cleanup(func() { pgRetrySleep = old })

	calls := 0
	err := retryPostgres(context.Background(), func(context.Context) error {
		calls++
		if calls < 4 {
			// Class 08 — Connection Exception (см. github.com/jackc/pgerrcode).
			return &pgconn.PgError{Code: pgerrcode.ConnectionFailure}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if calls != 4 {
		t.Fatalf("expected 4 op invocations, got %d", calls)
	}
}

func TestRetryPostgres_StopsOnUniqueViolation(t *testing.T) {
	old := pgRetrySleep
	pgRetrySleep = func(context.Context, time.Duration) error { return nil }
	t.Cleanup(func() { pgRetrySleep = old })

	calls := 0
	err := retryPostgres(context.Background(), func(context.Context) error {
		calls++
		return &pgconn.PgError{Code: pgerrcode.UniqueViolation}
	})
	if err == nil {
		t.Fatal("expected error")
	}
	if calls != 1 {
		t.Fatalf("expected single attempt for non-retriable SQLSTATE, got %d", calls)
	}
}

func TestRetryPostgres_ContextCanceledDuringBackoff(t *testing.T) {
	old := pgRetrySleep
	pgRetrySleep = sleepCtx
	t.Cleanup(func() { pgRetrySleep = old })

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := retryPostgres(ctx, func(context.Context) error {
		return &pgconn.PgError{Code: pgerrcode.ConnectionFailure}
	})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context.Canceled, got %v", err)
	}
}

func TestIsConnectionExceptionClass(t *testing.T) {
	if !isConnectionExceptionClass(pgerrcode.ProtocolViolation) {
		t.Fatal("08P01 should be class 08")
	}
	if isConnectionExceptionClass(pgerrcode.UniqueViolation) {
		t.Fatal("23505 must not be class 08")
	}
}
