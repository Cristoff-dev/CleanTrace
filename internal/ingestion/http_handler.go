package ingestion

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"runtime"
	"sync"
	"time"

	"github.com/Cristoff-dev/cleantrace/internal/core/domain"
	"github.com/Cristoff-dev/cleantrace/internal/core/ports"
)

// IncomingBatch represents the standard JSON payload structure expected from external collectors.
type IncomingBatch struct {
	Logs []domain.LogEntry `json:"logs"`
}

type HTTPHandler struct {
	masker ports.MaskerEngine
	repo   ports.LogRepository
}

// NewHTTPHandler injects the required application ports into the delivery mechanism.
func NewHTTPHandler(masker ports.MaskerEngine, repo ports.LogRepository) *HTTPHandler {
	return &HTTPHandler{
		masker: masker,
		repo:   repo,
	}
}

// CollectLogs handles the POST requests, sanitizes payloads concurrently, and dispatches them to storage.
func (h *HTTPHandler) CollectLogs(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var batch IncomingBatch
	if err := json.NewDecoder(r.Body).Decode(&batch); err != nil {
		http.Error(w, "Malformed JSON payload", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	batchSize := len(batch.Logs)
	if batchSize == 0 {
		w.WriteHeader(http.StatusAccepted)
		return
	}

	processedLogs := make([]*domain.LogEntry, batchSize)
	var wg sync.WaitGroup

	// Bounded concurrency using a semaphore pattern to prevent goroutine leaks on massive payloads.
	// Limits concurrent masking operations to the number of available CPU cores multiplied by 4.
	maxWorkers := runtime.NumCPU() * 4
	semaphore := make(chan struct{}, maxWorkers)

	for i := range batch.Logs {
		wg.Add(1)
		semaphore <- struct{}{}

		go func(idx int, entry domain.LogEntry) {
			defer wg.Done()
			defer func() { <-semaphore }()

			sanitized, wasMasked := h.masker.MaskSensitiveData(entry.RawPayload)

			entry.RawPayload = sanitized
			entry.MaskedData = wasMasked

			if !json.Valid([]byte(sanitized)) {
				wrapped, err := json.Marshal(map[string]string{"message": sanitized})
				if err == nil {
					sanitized = string(wrapped)
				}
			}

			entry.RawPayload = sanitized
			entry.MaskedData = wasMasked

			if entry.Timestamp.IsZero() {
				entry.Timestamp = time.Now().UTC()
			}

			processedLogs[idx] = &entry
		}(i, batch.Logs[i])
	}

	wg.Wait()

	// Asynchronous dispatch to the storage layer to ensure low-latency HTTP responses.
	// context.Background() is used because the HTTP request context will be canceled immediately after responding.
	go func(entries []*domain.LogEntry) {
		err := h.repo.SaveBatch(context.Background(), entries)
		if err != nil {
			fmt.Printf("❌ DB error critical: %v\n", err)
		} else {
			fmt.Println("✅ Batch successfully persisted to the database.")
		}
	}(processedLogs)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	_, _ = w.Write([]byte(`{"status": "accepted", "message": "logs processed and queued for persistence"}`))
}
