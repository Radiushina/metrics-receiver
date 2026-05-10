package repository

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
)

// Интервалы перед 2-й, 3-й и 4-й попыткой (до четырёх попыток на операцию).
var pgRetryDelays = []time.Duration{
	time.Second,
	3 * time.Second,
	5 * time.Second,
}

// pgRetrySleep подменяется в тестах.
var pgRetrySleep = sleepCtx

func sleepCtx(ctx context.Context, d time.Duration) error {
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-t.C:
		return nil
	}
}

// isConnectionExceptionClass возвращает true для SQLSTATE класса 08 — Connection Exception
// (см. константы в github.com/jackc/pgerrcode, например pgerrcode.ConnectionFailure).
func isConnectionExceptionClass(sqlState string) bool {
	if len(sqlState) < 2 {
		return false
	}
	return sqlState[0] == '0' && sqlState[1] == '8'
}

func isRetriablePostgresErr(err error) bool {
	if err == nil {
		return false
	}
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return isConnectionExceptionClass(pgErr.Code)
	}
	return false
}

// retryPostgres выполняет op до четырёх раз при ошибках соединения с БД (class 08).
func retryPostgres(ctx context.Context, op func(context.Context) error) error {
	var lastErr error
	for attempt := 0; attempt < 1+len(pgRetryDelays); attempt++ {
		if attempt > 0 {
			if err := pgRetrySleep(ctx, pgRetryDelays[attempt-1]); err != nil {
				return err
			}
		}
		err := op(ctx)
		if err == nil {
			return nil
		}
		lastErr = err
		if !isRetriablePostgresErr(err) {
			return err
		}
	}
	return lastErr
}
