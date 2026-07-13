// Package piidetector implements pattern-based PII detection and redaction.
package piidetector

import (
	"github.com/bzdvdn/draftrag/internal/domain"
)

const redactedMarker = "<redacted>"

// PIICategories задаёт набор категорий PII для встроенного детектора.
type PIICategories struct {
	Email      bool
	Phone      bool
	SSN        bool
	CreditCard bool
}

// @sk-task pii-guardrails#T1.2: CompositePIIDetector (DEC-002, AC-004)
//
// CompositePIIDetector последовательно применяет набор под-детекторов.
type CompositePIIDetector struct {
	detectors []domain.PIIDetector
}

// NewCompositePIIDetector создаёт композитный детектор из набора под-детекторов.
func NewCompositePIIDetector(detectors ...domain.PIIDetector) domain.PIIDetector {
	return &CompositePIIDetector{detectors: detectors}
}

// Detect последовательно применяет все под-детекторы.
func (d *CompositePIIDetector) Detect(text string) string {
	for _, det := range d.detectors {
		text = det.Detect(text)
	}
	return text
}

// NewDefaultPIIDetector создаёт CompositePIIDetector из встроенных детекторов
// согласно включённым категориям.
func NewDefaultPIIDetector(cats PIICategories) domain.PIIDetector {
	var dets []domain.PIIDetector
	if cats.Email {
		dets = append(dets, NewEmailDetector())
	}
	if cats.Phone {
		dets = append(dets, NewPhoneDetector())
	}
	if cats.SSN {
		dets = append(dets, NewSSNDetector())
	}
	if cats.CreditCard {
		dets = append(dets, NewCreditCardDetector())
	}
	return NewCompositePIIDetector(dets...)
}
