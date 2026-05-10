package repository

import (
	"context"
	"errors"
	"fmt"

	models "github.com/Radiushina/metrics-receiver.git/internal/model"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// PostgresRepo хранит метрики в PostgreSQL.
// Одна строка на имя метрики в таблицах gauges и counters.
type PostgresRepo struct {
	pool *pgxpool.Pool
}

// NewPostgresRepo создаёт репозиторий для работы с метриками через пул соединений pgx.
// Пул должен быть уже создан и проверен (Ping); миграции схемы вызываются отдельно.
func NewPostgresRepo(pool *pgxpool.Pool) *PostgresRepo {
	return &PostgresRepo{pool: pool}
}

// SetGauge записывает или обновляет gauge в таблице gauges.
func (r *PostgresRepo) SetGauge(ctx context.Context, name string, value float64) {
	const q = `
INSERT INTO gauges (name, value) VALUES ($1, $2)
ON CONFLICT (name) DO UPDATE SET value = EXCLUDED.value`
	_ = retryPostgres(ctx, func(ctx context.Context) error {
		_, err := r.pool.Exec(ctx, q, name, value)
		return err
	})
}

// AddCounter прибавляет delta к счётчику в таблице counters (или создаёт строку).
func (r *PostgresRepo) AddCounter(ctx context.Context, name string, delta int64) {
	const q = `
INSERT INTO counters (name, value) VALUES ($1, $2)
ON CONFLICT (name) DO UPDATE SET value = counters.value + EXCLUDED.value`
	_ = retryPostgres(ctx, func(ctx context.Context) error {
		_, err := r.pool.Exec(ctx, q, name, delta)
		return err
	})
}

// GetGauge возвращает значение gauge и false, если строки с таким именем нет.
func (r *PostgresRepo) GetGauge(ctx context.Context, name string) (float64, bool) {
	var v float64
	var found bool
	err := retryPostgres(ctx, func(ctx context.Context) error {
		err := r.pool.QueryRow(ctx, `SELECT value FROM gauges WHERE name = $1`, name).Scan(&v)
		if errors.Is(err, pgx.ErrNoRows) {
			found = false
			return nil
		}
		if err != nil {
			return err
		}
		found = true
		return nil
	})
	if err != nil {
		return 0, false
	}
	return v, found
}

// GetCounter возвращает значение счётчика и false, если строки с таким именем нет.
func (r *PostgresRepo) GetCounter(ctx context.Context, name string) (int64, bool) {
	var v int64
	var found bool
	err := retryPostgres(ctx, func(ctx context.Context) error {
		err := r.pool.QueryRow(ctx, `SELECT value FROM counters WHERE name = $1`, name).Scan(&v)
		if errors.Is(err, pgx.ErrNoRows) {
			found = false
			return nil
		}
		if err != nil {
			return err
		}
		found = true
		return nil
	})
	if err != nil {
		return 0, false
	}
	return v, found
}

// Gauges возвращает все gauge-метрики из таблицы gauges (или nil при ошибке запроса).
func (r *PostgresRepo) Gauges(ctx context.Context) map[string]float64 {
	var out map[string]float64
	err := retryPostgres(ctx, func(ctx context.Context) error {
		rows, err := r.pool.Query(ctx, `SELECT name, value FROM gauges`)
		if err != nil {
			return err
		}
		defer rows.Close()
		m := make(map[string]float64)
		for rows.Next() {
			var name string
			var v float64
			if err := rows.Scan(&name, &v); err != nil {
				return err
			}
			m[name] = v
		}
		if err := rows.Err(); err != nil {
			return err
		}
		out = m
		return nil
	})
	if err != nil {
		return nil
	}
	return out
}

// Counters возвращает все счётчики из таблицы counters (или nil при ошибке запроса).
func (r *PostgresRepo) Counters(ctx context.Context) map[string]int64 {
	var out map[string]int64
	err := retryPostgres(ctx, func(ctx context.Context) error {
		rows, err := r.pool.Query(ctx, `SELECT name, value FROM counters`)
		if err != nil {
			return err
		}
		defer rows.Close()
		m := make(map[string]int64)
		for rows.Next() {
			var name string
			var v int64
			if err := rows.Scan(&name, &v); err != nil {
				return err
			}
			m[name] = v
		}
		if err := rows.Err(); err != nil {
			return err
		}
		out = m
		return nil
	})
	if err != nil {
		return nil
	}
	return out
}

// UpdateMetricsBatch применяет список обновлений метрик атомарно.
// Для gauge выполняется upsert (запись абсолютного значения), для counter — инкремент на delta.
// Все операции выполняются в одной транзакции: при любой ошибке происходит откат,
// частичные обновления не сохраняются.
func (r *PostgresRepo) UpdateMetricsBatch(ctx context.Context, metrics []models.Metrics) error {
	if len(metrics) == 0 {
		return nil
	}
	return retryPostgres(ctx, func(ctx context.Context) error {
		return r.updateMetricsBatchOnce(ctx, metrics)
	})
}

func (r *PostgresRepo) updateMetricsBatchOnce(ctx context.Context, metrics []models.Metrics) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	// можно вызвать Rollback в defer,
	// если Commit будет раньше, то откат проигнорируется
	defer tx.Rollback(ctx)

	const qGauge = `
INSERT INTO gauges (name, value) VALUES ($1, $2)
ON CONFLICT (name) DO UPDATE SET value = EXCLUDED.value`

	const qCounter = `
INSERT INTO counters (name, value) VALUES ($1, $2)
ON CONFLICT (name) DO UPDATE SET value = counters.value + EXCLUDED.value`

	const stmtGauge = "upsert_gauge"
	const stmtCounter = "inc_counter"

	if _, err := tx.Prepare(ctx, stmtGauge, qGauge); err != nil {
		return err
	}
	if _, err := tx.Prepare(ctx, stmtCounter, qCounter); err != nil {
		return err
	}

	for _, m := range metrics {
		switch m.MType {
		case models.Gauge:
			if m.Value == nil {
				return fmt.Errorf("missing gauge value for %q", m.ID)
			}
			if _, err := tx.Exec(ctx, stmtGauge, m.ID, *m.Value); err != nil {
				return err
			}
		case models.Counter:
			if m.Delta == nil {
				return fmt.Errorf("missing counter delta for %q", m.ID)
			}
			if _, err := tx.Exec(ctx, stmtCounter, m.ID, *m.Delta); err != nil {
				return err
			}
		default:
			return fmt.Errorf("invalid metric type: %q", m.MType)
		}
	}

	return tx.Commit(ctx)
}
