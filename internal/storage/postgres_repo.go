package storage

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Cristoff-dev/cleantrace/internal/core/domain"
	"github.com/Cristoff-dev/cleantrace/internal/core/ports"
)

type postgresRepo struct {
	pool *pgxpool.Pool
}

// NewPostgresRepo initializes the PostgreSQL repository using a connection pool.
// The pgxpool handles connection multiplexing automatically for high concurrency.
func NewPostgresRepo(pool *pgxpool.Pool) ports.LogRepository {
	return &postgresRepo{
		pool: pool,
	}
}

// Save persists a single log entry. For mass ingestion, SaveBatch is preferred.
func (r *postgresRepo) Save(ctx context.Context, entry *domain.LogEntry) error {
	query := `
		INSERT INTO sanitized_logs (id, timestamp, environment, level, service, raw_payload, masked_data)
		VALUES ($1, $2, $3, $4, $5, $6::jsonb, $7)
	`
	_, err := r.pool.Exec(ctx, query,
		entry.ID,
		entry.Timestamp,
		entry.Environment,
		entry.Level,
		entry.Service,
		entry.RawPayload,
		entry.MaskedData,
	)
	if err != nil {
		return fmt.Errorf("failed to insert single log entry: %w", err)
	}
	return nil
}

// SaveBatch executes bulk inserts using pgx.Batch.
// This reduces TCP roundtrips to the database from O(N) to O(1), which is critical
// for avoiding database connection starvation during massive log ingestion spikes.
func (r *postgresRepo) SaveBatch(ctx context.Context, entries []*domain.LogEntry) error {
	if len(entries) == 0 {
		return nil
	}

	batch := &pgx.Batch{}
	query := `
		INSERT INTO sanitized_logs (id, timestamp, environment, level, service, raw_payload, masked_data)
		VALUES ($1, $2, $3, $4, $5, $6::jsonb, $7)
	`

	for _, entry := range entries {
		batch.Queue(query,
			entry.ID,
			entry.Timestamp,
			entry.Environment,
			entry.Level,
			entry.Service,
			entry.RawPayload,
			entry.MaskedData,
		)
	}

	br := r.pool.SendBatch(ctx, batch)
	defer br.Close()

	for i := 0; i < len(entries); i++ {
		_, err := br.Exec()
		if err != nil {
			return fmt.Errorf("batch execution failed at log index %d: %w", i, err)
		}
	}

	return nil
}
