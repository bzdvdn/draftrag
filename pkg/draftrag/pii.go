package draftrag

import (
	"github.com/bzdvdn/draftrag/internal/domain"
	"github.com/bzdvdn/draftrag/internal/infrastructure/piidetector"
)

// @sk-task pii-guardrails#T2.1: re-export PIIDetector (RQ-004, AC-005)
// PIIDetector — интерфейс для обнаружения и цензурирования PII в тексте.
type PIIDetector = domain.PIIDetector

// @sk-task pii-guardrails#T2.1: re-export PIICategories (RQ-006, AC-004)
// PIICategories задаёт набор категорий PII для встроенного детектора.
type PIICategories = piidetector.PIICategories

// NewDefaultPIIDetector создаёт составной PII-детектор из встроенных
// pattern-детекторов согласно включённым категориям.
func NewDefaultPIIDetector(cats PIICategories) PIIDetector {
	return piidetector.NewDefaultPIIDetector(piidetector.PIICategories(cats))
}

// NewCompositePIIDetector создаёт PII-детектор, последовательно применяющий
// переданные детекторы. Полезно для комбинации встроенных и кастомных детекторов.
func NewCompositePIIDetector(detectors ...PIIDetector) PIIDetector {
	return piidetector.NewCompositePIIDetector(detectors...)
}
