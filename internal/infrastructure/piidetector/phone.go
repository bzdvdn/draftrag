package piidetector

import (
	"regexp"

	"github.com/bzdvdn/draftrag/internal/domain"
)

// Телефон: E.164 (+1-555-123-4567), РФ/Европа (+7-900-123-45-67) и локальные форматы (555-123-4567).
// Три альтернативных формата:
// 1. E.164: +<country><number> с опциональными разделителями (последняя группа 4+ цифр)
// 2. US/локальный: 3-3-4 цифры (не пересекается с SSN 3-2-4)
// 3. E.164 multi-segment: +<country> с короткими группами (напр., +7-900-123-45-67, +49-30-123-456-78)
//
// @sk-task pii-guardrails#T5.1: добавлен паттерн для РФ/Европа (verify concern #1)
var phoneRe = regexp.MustCompile(
	`\+\d{1,3}[-.\s]?\(?\d{2,4}\)?[-.\s]?\d{2,4}[-.\s]?\d{4,}\b|` +
		`\b\d{3}[-.\s]?\d{3}[-.\s]?\d{4}\b|` +
		`\+\d{1,3}[-.\s]?\(?\d{2,4}\)?(?:[-.\s]?\d{2,4}){2,}\b`,
)

// @sk-task pii-guardrails#T1.2: PhoneDetector (RQ-003, AC-004)
type PhoneDetector struct{}

// NewPhoneDetector создаёт детектор телефонных номеров.
func NewPhoneDetector() domain.PIIDetector {
	return &PhoneDetector{}
}

// Detect заменяет телефонные номера на redacted marker.
func (d *PhoneDetector) Detect(text string) string {
	return phoneRe.ReplaceAllString(text, redactedMarker)
}
