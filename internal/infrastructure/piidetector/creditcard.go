package piidetector

import (
	"regexp"

	"github.com/bzdvdn/draftrag/internal/domain"
)

// Номер кредитной карты: 16 цифр, с пробелами или дефисами через каждые 4 цифры.
var creditCardRe = regexp.MustCompile(`\b\d{4}[-.\s]?\d{4}[-.\s]?\d{4}[-.\s]?\d{4}\b`)

// @sk-task pii-guardrails#T3.3: CreditCardDetector (RQ-003)
type CreditCardDetector struct{}

func NewCreditCardDetector() domain.PIIDetector {
	return &CreditCardDetector{}
}

func (d *CreditCardDetector) Detect(text string) string {
	return creditCardRe.ReplaceAllString(text, redactedMarker)
}
