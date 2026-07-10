package piidetector

import (
	"regexp"

	"github.com/bzdvdn/draftrag/internal/domain"
)

var ssnRe = regexp.MustCompile(`\b\d{3}-\d{2}-\d{4}\b`)

// @sk-task pii-guardrails#T1.2: SSNDetector (RQ-003, AC-004)
type SSNDetector struct{}

func NewSSNDetector() domain.PIIDetector {
	return &SSNDetector{}
}

func (d *SSNDetector) Detect(text string) string {
	return ssnRe.ReplaceAllString(text, redactedMarker)
}
