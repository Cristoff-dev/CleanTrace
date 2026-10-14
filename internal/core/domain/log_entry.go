package domain

import "time"

// LogLevel defines standard syslog severity levels.
type LogLevel string

const (
	InfoLevel  LogLevel = "INFO"
	WarnLevel  LogLevel = "WARN"
	ErrorLevel LogLevel = "ERROR"
	FatalLevel LogLevel = "FATAL"
)

// LogEntry represents the normalized data structure for downstream persistence.
type LogEntry struct {
	ID          string    `json:"id"`
	Timestamp   time.Time `json:"timestamp"`
	Environment string    `json:"environment"`
	Level       LogLevel  `json:"level"`
	Service     string    `json:"service"`
	RawPayload  string    `json:"raw_payload"`
	MaskedData  bool      `json:"masked_data"`
}