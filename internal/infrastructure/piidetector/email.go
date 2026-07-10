package piidetector

import (
	"regexp"

	"github.com/bzdvdn/draftrag/internal/domain"
)

var emailRe = regexp.MustCompile(`\b[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Za-z]{2,}\b`)

// @sk-task pii-guardrails#T1.2: EmailDetector (RQ-003, AC-004)
type EmailDetector struct{}

func NewEmailDetector() domain.PIIDetector {
	return &EmailDetector{}
}

func (d *EmailDetector) Detect(text string) string {
	return emailRe.ReplaceAllString(text, redactedMarker)
}
