package sanitizer

import (
	"regexp"

	"github.com/Cristoff-dev/cleantrace/internal/core/ports"
)

// PIIPattern holds the compiled regular expression and its redaction format.
type PIIPattern struct {
	Name        string
	Expression  *regexp.Regexp
	Replacement string
}

type regexMasker struct {
	patterns []PIIPattern
}

// NewRegexMasker initializes the engine, pre-compiling all RE2 expressions to guarantee O(n) execution time.
func NewRegexMasker() ports.MaskerEngine {
	return &regexMasker{
		patterns: []PIIPattern{
			{
				Name:        "CreditCard",
				Expression:  regexp.MustCompile(`(?:\d{4}[-\s]?){3}\d{4}`),
				Replacement: "****-****-****-****",
			},
			{
				Name:        "JWTToken",
				Expression:  regexp.MustCompile(`eyJ[A-Za-z0-9-_=]+\.[A-Za-z0-9-_=]+\.?[A-Za-z0-9-_.+/=]*`),
				Replacement: "[REDACTED_JWT]",
			},
			{
				Name:        "EmailAddress",
				Expression:  regexp.MustCompile(`[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}`),
				Replacement: "[REDACTED_EMAIL]",
			},
		},
	}
}

// MaskSensitiveData scans the payload and replaces sensitive matches.
func (rm *regexMasker) MaskSensitiveData(payload string) (string, bool) {
	if payload == "" {
		return payload, false
	}

	sanitizedPayload := payload
	wasMasked := false

	for _, pattern := range rm.patterns {
		if pattern.Expression.MatchString(sanitizedPayload) {
			sanitizedPayload = pattern.Expression.ReplaceAllString(sanitizedPayload, pattern.Replacement)
			wasMasked = true
		}
	}

	return sanitizedPayload, wasMasked
}
