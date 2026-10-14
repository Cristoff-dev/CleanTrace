package ports

import (
	"context"

	"github.com/Cristoff-dev/cleantrace/internal/core/domain"
)

// MaskerEngine defines the contract for Data Masking and PII obfuscation.
type MaskerEngine interface {
	MaskSensitiveData(payload string) (sanitizedPayload string, wasMasked bool)
}

// LogRepository defines the persistence contract for storage implementations.
type LogRepository interface {
	Save(ctx context.Context, entry *domain.LogEntry) error
	SaveBatch(ctx context.Context, entries []*domain.LogEntry) error
}
